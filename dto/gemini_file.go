package dto

// GeminiFileUploadResponse is the response returned by the Google Files API
// after a successful file upload.
type GeminiFileUploadResponse struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	URI         string `json:"uri"`
	MimeType    string `json:"mimeType,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty,string"`
	CreateTime  string `json:"createTime,omitempty"`
	ExpireTime  string `json:"expireTime,omitempty"`
	State       string `json:"state,omitempty"`
}
