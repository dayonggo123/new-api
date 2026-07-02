package omniflash

// OmniFlashImage represents an inline image input for Omni Flash interactions.
type OmniFlashImage struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded,omitempty"`
	MimeType           string `json:"mimeType,omitempty"`
}

// OmniFlashVideo represents a Google file URI video reference.
type OmniFlashVideo struct {
	URI string `json:"uri,omitempty"`
}

// OmniFlashReference represents an extension/reference video input.
type OmniFlashReference struct {
	Video *OmniFlashVideo `json:"video,omitempty"`
}

// OmniFlashInput represents one input item in the interaction.
type OmniFlashInput struct {
	Text      string              `json:"text,omitempty"`
	Image     *OmniFlashImage     `json:"image,omitempty"`
	Video     *OmniFlashVideo     `json:"video,omitempty"`
	Reference *OmniFlashReference `json:"reference,omitempty"`
}

// OmniFlashConfig holds generation parameters for Omni Flash.
type OmniFlashConfig struct {
	AspectRatio     string `json:"aspectRatio,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
}

// OmniFlashRequest is the request body for the Omni Flash interactions endpoint.
type OmniFlashRequest struct {
	Input  []OmniFlashInput `json:"input"`
	Config OmniFlashConfig  `json:"config,omitempty"`
}

// OmniFlashSubmitResponse is the immediate submit response containing the interaction name.
type OmniFlashSubmitResponse struct {
	Name string `json:"name"`
}

// OmniFlashResultVideo represents a generated video in the operation result.
type OmniFlashResultVideo struct {
	Video struct {
		URI string `json:"uri"`
	} `json:"video"`
}

// OmniFlashOperationResponse is the polled operation result.
type OmniFlashOperationResponse struct {
	Name     string `json:"name"`
	Done     bool   `json:"done"`
	Response struct {
		GeneratedVideos []OmniFlashResultVideo `json:"generatedVideos"`
	} `json:"response"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
