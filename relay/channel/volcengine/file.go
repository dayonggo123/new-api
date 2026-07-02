package volcengine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// VolcengineFileUploadResponse is the raw response from VolcEngine /api/v3/files.
type VolcengineFileUploadResponse struct {
	ID                string          `json:"id"`
	Object            string          `json:"object"`
	Purpose           string          `json:"purpose"`
	Filename          string          `json:"filename"`
	Bytes             int64           `json:"bytes"`
	MimeType          string          `json:"mime_type"`
	CreatedAt         int64           `json:"created_at"`
	ExpireAt          int64           `json:"expire_at"`
	Status            string          `json:"status"`
	URL               string          `json:"url,omitempty"`
	PreprocessConfigs json.RawMessage `json:"preprocess_configs,omitempty"`
}

// buildVolcengineFileUploadRequest builds a multipart/form-data body for VolcEngine /api/v3/files.
func buildVolcengineFileUploadRequest(c *gin.Context, request *dto.FileUploadRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// purpose
	if err := writer.WriteField("purpose", request.Purpose); err != nil {
		writer.Close()
		return nil, "", fmt.Errorf("failed to write purpose field: %w", err)
	}

	// url-based upload
	if request.URL != "" {
		if err := writer.WriteField("url", request.URL); err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to write url field: %w", err)
		}
	}

	// binary file upload
	if request.File != nil {
		file, err := request.File.Open()
		if err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to open uploaded file: %w", err)
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
			writer.Close()
			return nil, "", fmt.Errorf("failed to create file form part: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	// preprocess_configs
	if len(request.PreprocessConfigs) > 0 {
		if err := writer.WriteField("preprocess_configs", string(request.PreprocessConfigs)); err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to write preprocess_configs field: %w", err)
		}
	}

	// tos
	if len(request.TOS) > 0 {
		if err := writer.WriteField("tos", string(request.TOS)); err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to write tos field: %w", err)
		}
	}

	// expire_at
	if request.ExpireAt > 0 {
		if err := writer.WriteField("expire_at", fmt.Sprintf("%d", request.ExpireAt)); err != nil {
			writer.Close()
			return nil, "", fmt.Errorf("failed to write expire_at field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &body, writer.FormDataContentType(), nil
}

// parseVolcengineFileResponse parses VolcEngine /api/v3/files response and returns an
// OpenAI-compatible FileUploadResponse.
func parseVolcengineFileResponse(body []byte) (*dto.FileUploadResponse, error) {
	var volcResp VolcengineFileUploadResponse
	if err := common.Unmarshal(body, &volcResp); err != nil {
		return nil, fmt.Errorf("failed to parse volcengine file response: %w", err)
	}

	resp := &dto.FileUploadResponse{
		ID:                volcResp.ID,
		Object:            volcResp.Object,
		Purpose:           volcResp.Purpose,
		Filename:          volcResp.Filename,
		Bytes:             volcResp.Bytes,
		MimeType:          volcResp.MimeType,
		CreatedAt:         volcResp.CreatedAt,
		ExpireAt:          volcResp.ExpireAt,
		Status:            volcResp.Status,
		URL:               volcResp.URL,
		PreprocessConfigs: volcResp.PreprocessConfigs,
	}
	if resp.Object == "" {
		resp.Object = "file"
	}
	return resp, nil
}

// volcengineFileUploadHandler reads the upstream HTTP response and writes an OpenAI-compatible JSON body.
func volcengineFileUploadHandler(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	fileResp, err := parseVolcengineFileResponse(responseBody)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	jsonResponse, err := common.Marshal(fileResp)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err := c.Writer.Write(jsonResponse); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	usage := &dto.Usage{
		PromptTokens: 1,
		TotalTokens:  1,
	}
	return usage, nil
}

// ConvertFileRequest converts a dto.FileUploadRequest into the VolcEngine /api/v3/files request body.
func (a *Adaptor) ConvertFileRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.FileUploadRequest) (any, error) {
	if request == nil {
		return nil, errors.New("file upload request is nil")
	}
	body, contentType, err := buildVolcengineFileUploadRequest(c, request)
	if err != nil {
		return nil, err
	}
	c.Set("volcengine_file_upload_content_type", contentType)
	return body, nil
}

// DoFileResponse writes the VolcEngine file upload response back to the client.
func (a *Adaptor) DoFileResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	return volcengineFileUploadHandler(c, resp)
}
