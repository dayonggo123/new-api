package openai

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

func createTestMultipartFile(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
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

func TestConvertFileRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileHeader := createTestMultipartFile(t, "test.json", "application/json", []byte(`{"key":"value"}`))
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

	buf, ok := body.(*bytes.Buffer)
	require.True(t, ok)
	assert.True(t, bytes.Contains(buf.Bytes(), []byte(`form-data; name="file"; filename="test.json"`)))
	assert.True(t, bytes.Contains(buf.Bytes(), []byte(`{"key":"value"}`)))

	contentType, exists := c.Get(openaiFileUploadContentTypeContextKey)
	require.True(t, exists)
	assert.Contains(t, contentType.(string), "multipart/form-data")
}

func TestOaiFileHandler_UploadResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	respBody, err := json.Marshal(dto.FileUploadResponse{
		ID:            "file-abc123",
		Object:        "file",
		Purpose:       "assistants",
		Filename:      "test.json",
		Bytes:         1234,
		MimeType:      "application/json",
		CreatedAt:     1700000000,
		Status:        "processed",
		StatusDetails: "ok",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}
	info := &relaycommon.RelayInfo{}

	usage, apiErr := OaiFileHandler(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var uploadResp dto.FileUploadResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &uploadResp))
	assert.Equal(t, "file-abc123", uploadResp.ID)
	assert.Equal(t, "file", uploadResp.Object)
	assert.Equal(t, "assistants", uploadResp.Purpose)
	assert.Equal(t, "test.json", uploadResp.Filename)
	assert.Equal(t, int64(1234), uploadResp.Bytes)
	assert.Equal(t, "processed", uploadResp.Status)
	assert.Equal(t, "ok", uploadResp.StatusDetails)
}

func TestOaiFileHandler_ListResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	respBody, err := json.Marshal(dto.FileListResponse{
		Object: "list",
		Data: []dto.FileUploadResponse{
			{
				ID:       "file-abc123",
				Object:   "file",
				Filename: "test.json",
				Status:   "processed",
			},
		},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}
	info := &relaycommon.RelayInfo{}

	usage, apiErr := OaiFileHandler(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var listResp dto.FileListResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &listResp))
	assert.Equal(t, "list", listResp.Object)
	require.Len(t, listResp.Data, 1)
	assert.Equal(t, "file-abc123", listResp.Data[0].ID)
	assert.Equal(t, "processed", listResp.Data[0].Status)
}

func TestOaiFileHandler_DeleteResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	respBody, err := json.Marshal(dto.FileDeleteResponse{
		ID:      "file-abc123",
		Object:  "file",
		Deleted: true,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}
	info := &relaycommon.RelayInfo{}

	usage, apiErr := OaiFileHandler(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var deleteResp dto.FileDeleteResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &deleteResp))
	assert.Equal(t, "file-abc123", deleteResp.ID)
	assert.Equal(t, "file", deleteResp.Object)
	assert.True(t, deleteResp.Deleted)
}

func TestOaiFileHandler_RetrieveResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	respBody, err := json.Marshal(dto.FileUploadResponse{
		ID:        "file-abc123",
		Object:    "file",
		Purpose:   "assistants",
		Filename:  "test.json",
		Bytes:     1234,
		MimeType:  "application/json",
		CreatedAt: 1700000000,
		Status:    "processed",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}
	info := &relaycommon.RelayInfo{}

	usage, apiErr := OaiFileHandler(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var retrieveResp dto.FileUploadResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &retrieveResp))
	assert.Equal(t, "file-abc123", retrieveResp.ID)
	assert.Equal(t, "file", retrieveResp.Object)
	assert.Equal(t, "processed", retrieveResp.Status)
}

func TestConvertFileRequest_URLBasedUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	request := &dto.FileUploadRequest{
		URL:     "https://example.com/doc.pdf",
		Purpose: "assistants",
	}

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{}
	_, err := adaptor.ConvertFileRequest(c, info, request)
	require.Error(t, err)
}

func TestConvertFileRequest_DefaultContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileHeader := createTestMultipartFile(t, "test.bin", "", []byte(`binary`))
	require.Empty(t, fileHeader.Header.Get("Content-Type"))

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

	buf, ok := body.(*bytes.Buffer)
	require.True(t, ok)
	assert.True(t, bytes.Contains(buf.Bytes(), []byte(`application/octet-stream`)))
}
