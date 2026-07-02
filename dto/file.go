package dto

import (
	"encoding/json"
	"mime/multipart"

	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// FileUploadRequest represents a client request to the /v1/files endpoint.
// It supports both binary file upload and URL-based upload.
type FileUploadRequest struct {
	File              *multipart.FileHeader `json:"-"`
	URL               string                `json:"url,omitempty"`
	Purpose           string                `json:"purpose,omitempty"`
	PreprocessConfigs json.RawMessage       `json:"preprocess_configs,omitempty"`
	TOS               json.RawMessage       `json:"tos,omitempty"`
	ExpireAt          int64                 `json:"expire_at,omitempty"`
}

// FileUploadResponse represents the standard OpenAI-compatible file object
// returned by the /v1/files endpoint.
type FileUploadResponse struct {
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

func (f *FileUploadRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{}
}

func (f *FileUploadRequest) IsStream(c *gin.Context) bool {
	return false
}

func (f *FileUploadRequest) SetModelName(modelName string) {
}
