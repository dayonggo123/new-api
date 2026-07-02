package openai

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const openaiFileUploadContentTypeContextKey = "openai_file_upload_content_type"

// ConvertFileRequest converts an OpenAI-compatible file upload request into a
// multipart/form-data body suitable for upstream OpenAI /v1/files.
func (a *Adaptor) ConvertFileRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.FileUploadRequest) (any, error) {
	if request == nil {
		return nil, errors.New("file upload request is nil")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("purpose", request.Purpose); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("failed to write purpose field: %w", err)
	}

	if request.File != nil {
		file, err := request.File.Open()
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("failed to open uploaded file: %w", err)
		}
		defer file.Close()

		contentType := request.File.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, request.File.Filename))
		hdr.Set("Content-Type", contentType)

		part, err := writer.CreatePart(hdr)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("failed to create file form part: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	} else if request.URL != "" {
		if err := a.writeFileFromURL(writer, request.URL); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("failed to upload file from URL: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	c.Set(openaiFileUploadContentTypeContextKey, writer.FormDataContentType())
	return &body, nil
}

// DoFileResponse parses the upstream OpenAI file upload response and writes it
// back to the client.
func (a *Adaptor) DoFileResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	return OaiFileHandler(c, resp, info)
}

// OaiFileHandler parses OpenAI Files API responses for POST upload, GET list,
// GET retrieve and DELETE operations.
func OaiFileHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, service.RelayErrorHandler(c.Request.Context(), resp, false)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("[OaiFileHandler] upstream response body: %s", string(responseBody)))
	}

	var raw map[string]any
	if err := common.Unmarshal(responseBody, &raw); err != nil {
		// Fallback to raw passthrough if the body is not valid JSON.
		service.IOCopyBytesGracefully(c, resp, responseBody)
		return &dto.Usage{}, nil
	}

	objectVal, _ := raw["object"].(string)
	deletedVal := false
	if v, ok := raw["deleted"].(bool); ok {
		deletedVal = v
	}

	var output []byte
	switch {
	case objectVal == "list":
		var listResp dto.FileListResponse
		if err := common.Unmarshal(responseBody, &listResp); err != nil {
			service.IOCopyBytesGracefully(c, resp, responseBody)
			return &dto.Usage{}, nil
		}
		output, err = common.Marshal(listResp)
	case objectVal == "file" && deletedVal:
		var deleteResp dto.FileDeleteResponse
		if err := common.Unmarshal(responseBody, &deleteResp); err != nil {
			service.IOCopyBytesGracefully(c, resp, responseBody)
			return &dto.Usage{}, nil
		}
		output, err = common.Marshal(deleteResp)
	case objectVal == "file":
		var fileResp dto.FileUploadResponse
		if err := common.Unmarshal(responseBody, &fileResp); err != nil {
			service.IOCopyBytesGracefully(c, resp, responseBody)
			return &dto.Usage{}, nil
		}
		output, err = common.Marshal(fileResp)
	default:
		output = responseBody
	}

	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	service.IOCopyBytesGracefully(c, resp, output)
	return &dto.Usage{}, nil
}

// writeFileFromURL downloads the file at the given URL and writes it to the
// multipart writer as the "file" form part.
func (a *Adaptor) writeFileFromURL(writer *multipart.Writer, fileURL string) error {
	parsedURL, err := url.Parse(fileURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid file URL: %s", fileURL)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to download file from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file from URL: received status %d", resp.StatusCode)
	}

	filename := extractFilenameFromURL(parsedURL, resp)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	hdr.Set("Content-Type", contentType)

	part, err := writer.CreatePart(hdr)
	if err != nil {
		return fmt.Errorf("failed to create file form part: %w", err)
	}
	if _, err := io.Copy(part, resp.Body); err != nil {
		return fmt.Errorf("failed to copy downloaded file content: %w", err)
	}
	return nil
}

// extractFilenameFromURL derives a filename from the URL path or the
// Content-Disposition header, falling back to "upload.bin".
func extractFilenameFromURL(parsedURL *url.URL, resp *http.Response) string {
	if base := path.Base(parsedURL.Path); base != "" && base != "." {
		return base
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if filename := params["filename"]; filename != "" {
				return filename
			}
		}
	}
	return "upload.bin"
}
