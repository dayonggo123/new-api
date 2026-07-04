package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestBuildImageGenerationTaskResponse_IncludesTaskID(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_abc123",
		CreatedAt:  1715410000,
		SubmitTime: 1715410000,
		Status:     model.TaskStatusQueued,
		Progress:   "0%",
	}

	resp := buildImageGenerationTaskResponse(task)
	assert.Equal(t, "task_abc123", resp["id"])
	assert.Equal(t, "task_abc123", resp["task_id"])
	assert.Equal(t, "image.generation", resp["object"])
}

func TestBuildImageGenerationTaskResponse_CompletedTask(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_abc123",
		CreatedAt:  1715410000,
		SubmitTime: 1715410000,
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		FinishTime: 1715410015,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/image.png",
		},
		Data: []byte(`{"created":1715410000,"data":[{"url":"https://example.com/image.png"}]}`),
	}

	resp := buildImageGenerationTaskResponse(task)
	assert.Equal(t, "task_abc123", resp["id"])
	assert.Equal(t, "task_abc123", resp["task_id"])
	assert.Equal(t, "completed", resp["status"])
	assert.Equal(t, int64(1715410015), resp["completed_at"])

	metadata, ok := resp["metadata"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "https://example.com/image.png", metadata["url"])
}

func TestIsSyncImageAsyncChannel_Controller(t *testing.T) {
	// OpenAI-compatible channels should be treated as sync-to-async.
	openAICompatibleChannels := []int{
		constant.ChannelTypeOpenAI,
		constant.ChannelTypeOpenAIMax,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypeXinference,
		constant.ChannelTypeCustom,
		constant.ChannelTypeGemini,
		constant.ChannelTypeVolcEngine,
	}
	for _, channelType := range openAICompatibleChannels {
		assert.True(t, isSyncImageAsyncChannel(channelType), "channel type %d should be sync-to-async", channelType)
	}

	// Task image channels should NOT be treated as sync-to-async.
	taskImageChannels := []int{
		constant.ChannelTypeAPIMart,
		constant.ChannelTypeDuoYuanTanSuo,
		constant.ChannelTypeZhangyuge,
		constant.ChannelTypeVeo,
	}
	for _, channelType := range taskImageChannels {
		assert.False(t, isSyncImageAsyncChannel(channelType), "channel type %d should not be sync-to-async", channelType)
	}
}

func TestIsImageGenerationModel_GPTImage2(t *testing.T) {
	assert.True(t, common.IsImageGenerationModel("gpt-image-2"), "gpt-image-2 should be recognized as an image generation model")
	assert.True(t, common.IsImageGenerationModel("gpt-image-1"), "gpt-image-1 should be recognized as an image generation model")
	assert.False(t, common.IsImageGenerationModel("gpt-4o"), "gpt-4o should not be recognized as an image generation model")
}
