package dto

// GeminiImageConfig represents the responseFormat.image configuration used by
// native Gemini image generation models (e.g. gemini-3.1-flash-image).
type GeminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

// GeminiImageInlineData is the inline image data returned in a
// generateContent response for native image generation models.
type GeminiImageInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiImageGenerationCandidate mirrors a generateContent candidate that may
// contain both TEXT and IMAGE parts.
type GeminiImageGenerationCandidate struct {
	Index   int64            `json:"index"`
	Content GeminiChatContent `json:"content"`
}

// GeminiImageGenerationResponse is the response shape returned by Gemini
// generateContent when responseModalities includes IMAGE.
type GeminiImageGenerationResponse struct {
	Candidates    []GeminiImageGenerationCandidate `json:"candidates"`
	UsageMetadata GeminiUsageMetadata              `json:"usageMetadata,omitempty"`
}
