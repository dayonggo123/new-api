package dto

import (
	"strconv"
	"strings"
)

const (
	VideoStatusUnknown    = "unknown"
	VideoStatusQueued     = "queued"
	VideoStatusInProgress = "in_progress"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"
)

type OpenAIVideo struct {
	ID                 string            `json:"id"`
	UUID               string            `json:"uuid,omitempty"` // upstream uuid
	TaskID             string            `json:"task_id,omitempty"` //兼容旧接口 待废弃
	Object             string            `json:"object"`
	Model              string            `json:"model"`
	Status             string            `json:"status"` // Should use VideoStatus constants: VideoStatusQueued, VideoStatusInProgress, VideoStatusCompleted, VideoStatusFailed
	Progress           int               `json:"progress"`
	CreatedAt          int64             `json:"created_at"`
	CompletedAt        int64             `json:"completed_at,omitempty"`
	ExpiresAt          int64             `json:"expires_at,omitempty"`
	Seconds            string            `json:"seconds,omitempty"`
	Size               string            `json:"size,omitempty"`
	RemixedFromVideoID string            `json:"remixed_from_video_id,omitempty"`
	Error              *OpenAIVideoError `json:"error,omitempty"`
	Metadata           map[string]any    `json:"metadata,omitempty"`
}

func (m *OpenAIVideo) SetProgressStr(progress string) {
	progress = strings.TrimSuffix(progress, "%")
	m.Progress, _ = strconv.Atoi(progress)
}
func (m *OpenAIVideo) SetMetadata(k string, v any) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[k] = v
}
func NewOpenAIVideo() *OpenAIVideo {
	return &OpenAIVideo{
		Object: "video",
		Status: VideoStatusQueued,
	}
}

type OpenAIVideoError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// OpenAIVideoGenerationItem mirrors the OpenAI-compatible video generation
// result item returned by /v1/videos/{task_id}.
type OpenAIVideoGenerationItem struct {
	URL           string `json:"url"`
	LastFrameURL  string `json:"last_frame_url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// OpenAIVideoGenerationResponse mirrors the OpenAI-compatible video generation
// result response returned by /v1/videos/{task_id}.
type OpenAIVideoGenerationResponse struct {
	Created int64                       `json:"created"`
	Data    []OpenAIVideoGenerationItem `json:"data"`
	Usage   *Usage                      `json:"usage,omitempty"`
}
