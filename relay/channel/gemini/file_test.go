package gemini

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertFileRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileHeader := createTestFileHeader(t, "test.txt", "text/plain", []byte("hello world"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	request := &dto.FileUploadRequest{
		File:    fileHeader,
		Purpose: "assistants",
	}

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{}
	body, err := adaptor.ConvertFileRequest(c, info, request)
	require.NoError(t, err)
	require.NotNil(t, body)

	meta, ok := c.Get(geminiFileUploadMetaContextKey)
	require.True(t, ok)
	uploadMeta := meta.(*geminiFileUploadMeta)
	assert.Equal(t, "test.txt", uploadMeta.DisplayName)
	assert.Equal(t, "text/plain", uploadMeta.MimeType)
	assert.Equal(t, int64(11), uploadMeta.Size)
	assert.False(t, uploadMeta.IsResumable)
}

func TestDoFileResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	respBody, err := json.Marshal(dto.GeminiFileUploadResponse{
		Name:        "files/abc123",
		DisplayName: "uploaded_file",
		URI:         "https://generativelanguage.googleapis.com/files/abc123",
		MimeType:    "application/pdf",
		SizeBytes:   1024,
		CreateTime:  "2025-01-01T00:00:00Z",
		ExpireTime:  "2025-01-08T00:00:00Z",
		State:       "ACTIVE",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}
	info := &relaycommon.RelayInfo{
		Request: &dto.FileUploadRequest{Purpose: "assistants"},
	}

	adaptor := &Adaptor{}
	usage, apiErr := adaptor.DoFileResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var uploadResp dto.FileUploadResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &uploadResp))
	assert.Equal(t, "files/abc123", uploadResp.ID)
	assert.Equal(t, "uploaded_file", uploadResp.Filename)
	assert.Equal(t, "application/pdf", uploadResp.MimeType)
	assert.Equal(t, int64(1024), uploadResp.Bytes)
	assert.Equal(t, "assistants", uploadResp.Purpose)
	assert.Equal(t, "ACTIVE", uploadResp.Status)
	assert.Contains(t, uploadResp.URL, "files/abc123")
}

func createTestFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data) + 1024))
	require.NoError(t, err)
	files := form.File["file"]
	require.Len(t, files, 1)
	files[0].Header.Set("Content-Type", contentType)
	return files[0]
}
