package volcengine

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
)

func TestOaiImage2VolcengineImageRequest(t *testing.T) {
	n := uint(2)
	req := &dto.ImageRequest{
		Model:          "Doubao-Seedream-5.0-lite",
		Prompt:         "a cute cat",
		N:              &n,
		Size:           "2K",
		ResponseFormat: "url",
		Extra: map[string]json.RawMessage{
			"sequential_image_generation":        json.RawMessage(`"auto"`),
			"sequential_image_generation_options": json.RawMessage(`{"max_images":4}`),
			"output_format":                       json.RawMessage(`"png"`),
		},
	}

	volcReq, err := oaiImage2VolcengineImageRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, "Doubao-Seedream-5.0-lite", volcReq.Model)
	assert.Equal(t, "2K", volcReq.Size)
	assert.Equal(t, "auto", volcReq.SequentialImageGeneration)
	assert.Equal(t, json.RawMessage(`{"max_images":4}`), volcReq.SequentialImageGenerationOptions)
	assert.Equal(t, json.RawMessage(`"png"`), volcReq.OutputFormat)
	assert.Equal(t, uint(2), *volcReq.N)
}

func TestOaiImage2VolcengineImageRequestWithImageArray(t *testing.T) {
	req := &dto.ImageRequest{
		Model:  "Doubao-Seedream-4.5",
		Prompt: "merge styles",
		Image:  json.RawMessage(`["https://example.com/a.png","https://example.com/b.png"]`),
	}

	volcReq, err := oaiImage2VolcengineImageRequest(req)
	assert.NoError(t, err)
	images, ok := volcReq.Image.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"https://example.com/a.png", "https://example.com/b.png"}, images)
}

func TestConvertOpenAIRequestMessagesWithMultimodal(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "doubao-seed-1-6-thinking-250715",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "describe this"},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/image.png"}},
					map[string]any{"type": "input_audio", "input_audio": map[string]any{"data": "base64data", "format": "mp3"}},
					map[string]any{"type": "file", "file": map[string]any{"file_id": "file-123"}},
				},
			},
		},
	}

	err := convertOpenAIRequestMessages(nil, req)
	assert.NoError(t, err)
	assert.Len(t, req.Messages, 1)

	content, ok := req.Messages[0].Content.([]any)
	assert.True(t, ok)
	assert.Len(t, content, 4)

	imagePart := content[1].(map[string]any)
	assert.Equal(t, "image_url", imagePart["type"])
	assert.Equal(t, map[string]any{"url": "https://example.com/image.png"}, imagePart["image_url"])

	audioPart := content[2].(map[string]any)
	assert.Equal(t, "input_audio", audioPart["type"])
	audioData := audioPart["input_audio"].(map[string]any)
	assert.Equal(t, "data:audio/mp3;base64,base64data", audioData["data"])
	assert.Equal(t, "mp3", audioData["format"])

	filePart := content[3].(map[string]any)
	assert.Equal(t, "file", filePart["type"])
	assert.Equal(t, map[string]any{"file_id": "file-123"}, filePart["file"])
}

func TestConvertOpenAIRequestMessagesWithFileIDImage(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "doubao-seed-1-6-thinking-250715",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "file-abc123"}},
				},
			},
		},
	}

	err := convertOpenAIRequestMessages(nil, req)
	assert.NoError(t, err)

	content := req.Messages[0].Content.([]any)
	imagePart := content[0].(map[string]any)
	assert.Equal(t, map[string]any{"file_id": "file-abc123"}, imagePart["image_url"])
}

func TestParseVolcengineFileResponse(t *testing.T) {
	body := []byte(`{
		"id": "file-20251018114827-6zgrb",
		"object": "file",
		"purpose": "user_data",
		"filename": "demo.mp4",
		"bytes": 695110,
		"mime_type": "video/mp4",
		"created_at": 1760759307,
		"expire_at": 1761364107,
		"status": "processing"
	}`)

	resp, err := parseVolcengineFileResponse(body)
	assert.NoError(t, err)
	assert.Equal(t, "file-20251018114827-6zgrb", resp.ID)
	assert.Equal(t, "demo.mp4", resp.Filename)
	assert.Equal(t, "video/mp4", resp.MimeType)
	assert.Equal(t, "processing", resp.Status)
}
