package volcenginevideo

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/pkg/errors"
)

// ConvertToOpenAIVideo converts the persisted upstream task data into an
// OpenAI-compatible video generation result response.
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var result VolcengineVideoTaskResult
	if err := common.Unmarshal(originTask.Data, &result); err != nil {
		return nil, errors.Wrap(err, "unmarshal volcengine video task data failed")
	}

	resp := &dto.OpenAIVideoGenerationResponse{
		Created: originTask.CreatedAt,
		Data:    []dto.OpenAIVideoGenerationItem{},
	}

	// Prefer the upstream created_at timestamp if available.
	if result.CreatedAt > 0 {
		resp.Created = result.CreatedAt
	}

	videoURL := result.Content.VideoURL
	if videoURL == "" {
		videoURL = originTask.GetResultURL()
	}

	if videoURL != "" {
		item := dto.OpenAIVideoGenerationItem{
			URL:           videoURL,
			LastFrameURL:  result.Content.LastFrameURL,
			RevisedPrompt: originTask.Properties.Input,
		}
		resp.Data = append(resp.Data, item)
	}

	if result.Usage.CompletionTokens > 0 || result.Usage.TotalTokens > 0 {
		resp.Usage = &dto.Usage{
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}

	return common.Marshal(resp)
}
