package volcenginevideo

// VolcengineVideoRequest is the payload sent to the VolcEngine Ark video
// generation endpoint: POST /api/v3/contents/generations/tasks.
type VolcengineVideoRequest struct {
	Model         string                  `json:"model"`
	Content       []VolcengineContentItem `json:"content"`
	GenerateAudio bool                    `json:"generate_audio"`
	Ratio         string                  `json:"ratio,omitempty"`
	Duration      int                     `json:"duration,omitempty"`
	Resolution    string                  `json:"resolution,omitempty"`
	Watermark     bool                    `json:"watermark,omitempty"`
	Seed          int                     `json:"seed,omitempty"`
	CameraFixed   bool                    `json:"camera_fixed,omitempty"`
	Frames        int                     `json:"frames,omitempty"`
	Priority      string                  `json:"priority,omitempty"`
	ServiceTier   string                  `json:"service_tier,omitempty"`
	CallbackURL   string                  `json:"callback_url,omitempty"`
}

// VolcengineContentItem represents one entry in the content array. Only one of
// Text / ImageURL / VideoURL / AudioURL should be populated per item.
type VolcengineContentItem struct {
	Type     string    `json:"type,omitempty"`
	Role     string    `json:"role,omitempty"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
}

// MediaURL wraps a URL so it can be omitted when empty.
type MediaURL struct {
	URL string `json:"url,omitempty"`
}
