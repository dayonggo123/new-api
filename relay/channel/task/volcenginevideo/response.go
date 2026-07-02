package volcenginevideo

// VolcengineVideoSubmitResponse is the response returned when creating a task.
type VolcengineVideoSubmitResponse struct {
	ID string `json:"id"`
}

// VolcengineVideoTaskResult is the response returned when querying a task.
type VolcengineVideoTaskResult struct {
	ID      string                `json:"id"`
	Model   string                `json:"model"`
	Status  string                `json:"status"`
	Content VolcengineVideoContent `json:"content"`
	Usage   VolcengineUsage        `json:"usage"`
	Error   VolcengineError        `json:"error"`
	CreatedAt int64               `json:"created_at"`
	UpdatedAt int64               `json:"updated_at"`
}

// VolcengineVideoContent holds the generated video URL and its last frame.
type VolcengineVideoContent struct {
	VideoURL     string `json:"video_url"`
	LastFrameURL string `json:"last_frame_url,omitempty"`
}

// VolcengineUsage reports token consumption for billing.
type VolcengineUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// VolcengineError captures upstream failure details.
type VolcengineError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
