package gemini

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	// geminiFileUploadSimpleMaxSize is the threshold below which a simple
	// multipart upload is used (100 MB).
	geminiFileUploadSimpleMaxSize = 100 * 1024 * 1024
	// geminiFileUploadPDFResumableSize is the threshold for PDF files above
	// which resumable upload is used (50 MB).
	geminiFileUploadPDFResumableSize = 50 * 1024 * 1024
	// geminiFileUploadMaxSize is the maximum allowed file size (2 GB).
	geminiFileUploadMaxSize = 2 * 1024 * 1024 * 1024
	// geminiFileUploadURLMaxSize is the maximum size downloaded from a URL.
	geminiFileUploadURLMaxSize = 100 * 1024 * 1024
)

const geminiFileUploadContentTypeContextKey = "gemini_file_upload_content_type"
const geminiFileUploadMetaContextKey = "gemini_file_upload_meta"
const geminiFileUploadBodyContextKey = "gemini_file_upload_body"

// geminiFileUploadMeta holds the metadata required to upload a file to Google
// Files API.
type geminiFileUploadMeta struct {
	DisplayName string
	MimeType    string
	Size        int64
	Body        []byte
	IsResumable bool
}

// ConvertFileRequest converts an OpenAI file upload request into a Google Files
// API upload payload. The actual upload is performed by DoRequest.
func (a *Adaptor) ConvertFileRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.FileUploadRequest) (any, error) {
	meta, err := buildGeminiFileUploadMeta(c, request)
	if err != nil {
		return nil, err
	}
	if meta.Size > geminiFileUploadMaxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", meta.Size, geminiFileUploadMaxSize)
	}

	meta.IsResumable = shouldUseResumableUpload(meta.Size, meta.MimeType)
	c.Set(geminiFileUploadMetaContextKey, meta)

	if meta.IsResumable {
		// For resumable upload the body is uploaded in a separate PUT; return an
		// empty reader here.
		c.Set(geminiFileUploadContentTypeContextKey, "application/json")
		return bytes.NewReader([]byte{}), nil
	}

	body, contentType, err := buildGeminiMultipartUploadBody(meta)
	if err != nil {
		return nil, err
	}
	c.Set(geminiFileUploadContentTypeContextKey, contentType)
	c.Set(geminiFileUploadBodyContextKey, body)
	return bytes.NewReader(body), nil
}

// DoFileResponse parses the Google Files API response and returns an OpenAI
// compatible file object.
func (a *Adaptor) DoFileResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, apiErr *types.NewAPIError) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, service.RelayErrorHandler(c.Request.Context(), resp, false)
	}

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewOpenAIError(readErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var geminiResp dto.GeminiFileUploadResponse
	if err := common.Unmarshal(responseBody, &geminiResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	createdAt := parseGeminiTime(geminiResp.CreateTime)
	expireAt := parseGeminiTime(geminiResp.ExpireTime)

	openAIResp := dto.FileUploadResponse{
		ID:       geminiResp.Name,
		Object:   "file",
		Filename: geminiResp.DisplayName,
		Bytes:    geminiResp.SizeBytes,
		MimeType: geminiResp.MimeType,
		CreatedAt: createdAt,
		ExpireAt:  expireAt,
		Status:   geminiResp.State,
		URL:      geminiResp.URI,
	}
	if req, ok := info.Request.(*dto.FileUploadRequest); ok && req != nil {
		openAIResp.Purpose = req.Purpose
	}
	if openAIResp.Purpose == "" {
		openAIResp.Purpose = "assistants"
	}

	jsonResponse, marshalErr := common.Marshal(openAIResp)
	if marshalErr != nil {
		return nil, types.NewError(marshalErr, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)

	return &dto.Usage{}, nil
}

// doGeminiFileUploadRequest performs the Google Files API upload. It handles
// both simple multipart upload and resumable upload.
func doGeminiFileUploadRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	meta, ok := c.Get(geminiFileUploadMetaContextKey)
	if !ok {
		return nil, fmt.Errorf("gemini file upload metadata not found")
	}
	uploadMeta := meta.(*geminiFileUploadMeta)

	baseURL := info.ChannelBaseUrl
	version := model_setting.GetGeminiVersionSetting("default")
	uploadURL := fmt.Sprintf("%s/upload/%s/files", baseURL, version)

	if uploadMeta.IsResumable {
		return doGeminiResumableUpload(c, uploadMeta, uploadURL, info.ApiKey)
	}

	body, ok := c.Get(geminiFileUploadBodyContextKey)
	if !ok {
		// Fallback to requestBody if body was not stored in context.
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, requestBody); err != nil {
			return nil, fmt.Errorf("read file upload body failed: %w", err)
		}
		body = buf.Bytes()
	}
	bodyBytes := body.([]byte)

	contentType := ""
	if ct, ok := c.Get(geminiFileUploadContentTypeContextKey); ok {
		contentType = ct.(string)
	}
	if contentType == "" {
		contentType = fmt.Sprintf("multipart/form-data; boundary=%s", defaultMultipartBoundary)
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, uploadURL+"?uploadType=multipart", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("x-goog-api-key", info.ApiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	return doGeminiUploadHTTPRequest(c, req, info)
}

// handleGeminiFileUploadResponse is a thin wrapper used by Adaptor.DoResponse.
func handleGeminiFileUploadResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) *types.NewAPIError {
	_, err := new(Adaptor).DoFileResponse(c, resp, info)
	return err
}

// buildGeminiFileUploadMeta reads the file from the request (multipart or URL)
// and returns upload metadata.
func buildGeminiFileUploadMeta(c *gin.Context, request *dto.FileUploadRequest) (*geminiFileUploadMeta, error) {
	meta := &geminiFileUploadMeta{}

	if request.File != nil && request.File.Size > 0 {
		file, err := request.File.Open()
		if err != nil {
			return nil, fmt.Errorf("open file failed: %w", err)
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("read file failed: %w", err)
		}
		meta.Body = body
		meta.Size = int64(len(body))
		meta.MimeType = request.File.Header.Get("Content-Type")
		if meta.MimeType == "" || meta.MimeType == "application/octet-stream" {
			meta.MimeType = http.DetectContentType(body)
		}
		meta.DisplayName = request.File.Filename
		if meta.DisplayName == "" {
			meta.DisplayName = "uploaded_file"
		}
		return meta, nil
	}

	if request.URL != "" {
		body, mimeType, err := readURLBody(c, request.URL, geminiFileUploadURLMaxSize)
		if err != nil {
			return nil, fmt.Errorf("download url failed: %w", err)
		}
		meta.Body = body
		meta.Size = int64(len(body))
		meta.MimeType = mimeType
		meta.DisplayName = "uploaded_file"
		return meta, nil
	}

	return nil, fmt.Errorf("no file or url provided")
}

// shouldUseResumableUpload returns true for files >= 100 MB or PDF files > 50 MB.
func shouldUseResumableUpload(size int64, mimeType string) bool {
	if size >= geminiFileUploadSimpleMaxSize {
		return true
	}
	if strings.ToLower(mimeType) == "application/pdf" && size > geminiFileUploadPDFResumableSize {
		return true
	}
	return false
}

const defaultMultipartBoundary = "----NewAPIGeminiFileUploadBoundary"

// buildGeminiMultipartUploadBody builds a multipart/form-data body containing
// the file metadata and the file bytes.
func buildGeminiMultipartUploadBody(meta *geminiFileUploadMeta) ([]byte, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.SetBoundary(defaultMultipartBoundary); err != nil {
		return nil, "", fmt.Errorf("set multipart boundary failed: %w", err)
	}

	metadata := map[string]any{
		"file": map[string]any{
			"display_name": meta.DisplayName,
		},
	}
	metadataBytes, err := common.Marshal(metadata)
	if err != nil {
		return nil, "", fmt.Errorf("marshal metadata failed: %w", err)
	}
	part, err := writer.CreateFormField("metadata")
	if err != nil {
		return nil, "", fmt.Errorf("create metadata field failed: %w", err)
	}
	if _, err := part.Write(metadataBytes); err != nil {
		return nil, "", fmt.Errorf("write metadata failed: %w", err)
	}

	filePart, err := writer.CreateFormFile("file", meta.DisplayName)
	if err != nil {
		return nil, "", fmt.Errorf("create file field failed: %w", err)
	}
	if _, err := filePart.Write(meta.Body); err != nil {
		return nil, "", fmt.Errorf("write file field failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer failed: %w", err)
	}

	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", defaultMultipartBoundary)
	return body.Bytes(), contentType, nil
}

// doGeminiResumableUpload performs a two-step resumable upload: start session,
// then upload the bytes in a single chunk.
func doGeminiResumableUpload(c *gin.Context, meta *geminiFileUploadMeta, uploadURL, apiKey string) (*http.Response, error) {
	metadata := map[string]any{
		"file": map[string]any{
			"display_name": meta.DisplayName,
		},
	}
	metadataBytes, err := common.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata failed: %w", err)
	}

	startReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, uploadURL+"?uploadType=resumable", bytes.NewReader(metadataBytes))
	if err != nil {
		return nil, fmt.Errorf("new resumable start request failed: %w", err)
	}
	startReq.Header.Set("x-goog-api-key", apiKey)
	startReq.Header.Set("Content-Type", "application/json")
	startReq.Header.Set("X-Goog-Upload-Protocol", "resumable")
	startReq.Header.Set("X-Goog-Upload-Command", "start")
	startReq.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.FormatInt(meta.Size, 10))
	startReq.Header.Set("X-Goog-Upload-Header-Content-Type", meta.MimeType)
	startReq.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, fmt.Errorf("new http client failed: %w", err)
	}
	startResp, err := client.Do(startReq)
	if err != nil {
		return nil, fmt.Errorf("resumable start request failed: %w", err)
	}
	defer startResp.Body.Close()

	if startResp.StatusCode != http.StatusOK && startResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(startResp.Body)
		return nil, fmt.Errorf("resumable start failed with status %d: %s", startResp.StatusCode, string(body))
	}

	uploadLocation := startResp.Header.Get("X-Goog-Upload-URL")
	if uploadLocation == "" {
		return nil, fmt.Errorf("resumable upload url not found")
	}

	uploadReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPut, uploadLocation, bytes.NewReader(meta.Body))
	if err != nil {
		return nil, fmt.Errorf("new resumable upload request failed: %w", err)
	}
	uploadReq.Header.Set("Content-Type", meta.MimeType)
	uploadReq.Header.Set("X-Goog-Upload-Protocol", "resumable")
	uploadReq.Header.Set("X-Goog-Upload-Command", "upload,finalize")
	uploadReq.Header.Set("Content-Length", strconv.FormatInt(meta.Size, 10))

	uploadResp, err := client.Do(uploadReq)
	if err != nil {
		return nil, fmt.Errorf("resumable upload failed: %w", err)
	}
	return uploadResp, nil
}

// doGeminiUploadHTTPRequest sends the upload request through the common HTTP
// client configured for the channel.
func doGeminiUploadHTTPRequest(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) (*http.Response, error) {
	if info.ChannelSetting.Proxy != "" {
		client, err := service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return nil, fmt.Errorf("new proxy http client failed: %w", err)
		}
		return client.Do(req)
	}
	return service.GetHttpClient().Do(req)
}

// parseGeminiTime parses a Google timestamp into Unix seconds.
func parseGeminiTime(value string) int64 {
	if value == "" {
		return 0
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05.999Z"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// readURLBody downloads the content at url up to maxBytes and returns the bytes
// and the detected MIME type.
func readURLBody(c *gin.Context, url string, maxBytes int64) ([]byte, string, error) {
	client, err := service.GetHttpClientWithProxy("")
	if err != nil {
		return nil, "", fmt.Errorf("new http client failed: %w", err)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("new request failed: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download url failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download url returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read url body failed: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", fmt.Errorf("url body exceeds maximum size %d", maxBytes)
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(body)
	}
	return body, mimeType, nil
}
