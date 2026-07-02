package dto

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesRequest_ParseInput_String(t *testing.T) {
	raw := []byte(`{"model":"gpt-4.1","input":"hello world"}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	inputs := req.ParseInput()
	require.Len(t, inputs, 1)
	assert.Equal(t, "input_text", inputs[0].Type)
	assert.Equal(t, "hello world", inputs[0].Text)
}

func TestOpenAIResponsesRequest_ParseInput_Multimodal(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": [
					{"type": "input_text", "text": "describe this image"},
					{"type": "input_image", "image_url": "https://example.com/image.png"},
					{"type": "input_file", "file_url": "https://example.com/doc.pdf"}
				]
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	inputs := req.ParseInput()
	require.Len(t, inputs, 3)
	assert.Equal(t, "input_text", inputs[0].Type)
	assert.Equal(t, "describe this image", inputs[0].Text)
	assert.Equal(t, "input_image", inputs[1].Type)
	assert.Equal(t, "https://example.com/image.png", inputs[1].ImageUrl)
	assert.Equal(t, "input_file", inputs[2].Type)
	assert.Equal(t, "https://example.com/doc.pdf", inputs[2].FileUrl)
}

func TestOpenAIResponsesRequest_ParseInput_ImageBase64(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": [
					{"type": "input_image", "image_url": "data:image/png;base64,abc123"}
				]
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	inputs := req.ParseInput()
	require.Len(t, inputs, 1)
	assert.Equal(t, "input_image", inputs[0].Type)
	assert.Equal(t, "data:image/png;base64,abc123", inputs[0].ImageUrl)
}

func TestOpenAIResponsesRequest_ParseInput_ImageDetailPassThrough(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": [
					{"type": "input_image", "image_url": {"url": "https://example.com/image.png", "detail": "low"}}
				]
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	inputs := req.ParseInput()
	require.Len(t, inputs, 1)
	assert.Equal(t, "input_image", inputs[0].Type)
	assert.Equal(t, "https://example.com/image.png", inputs[0].ImageUrl)
	assert.Equal(t, "low", inputs[0].Detail)
}

func TestOpenAIResponsesRequest_ParseInput_StringContent(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": "hello world"
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	inputs := req.ParseInput()
	require.Len(t, inputs, 1)
	assert.Equal(t, "input_text", inputs[0].Type)
	assert.Equal(t, "hello world", inputs[0].Text)
}

func TestOpenAIResponsesRequest_GetTokenCountMeta_WithFiles(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": [
					{"type": "input_image", "image_url": "https://example.com/image.png"}
				]
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	meta := req.GetTokenCountMeta()
	require.Len(t, meta.Files, 1)
	assert.Equal(t, types.FileTypeImage, meta.Files[0].FileType)
}

func TestOpenAIResponsesRequest_GetTokenCountMeta_WithFileDetail(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-4.1",
		"input": [
			{
				"role": "user",
				"content": [
					{"type": "input_image", "image_url": {"url": "https://example.com/image.png", "detail": "low"}}
				]
			}
		]
	}`)
	var req OpenAIResponsesRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	meta := req.GetTokenCountMeta()
	require.Len(t, meta.Files, 1)
	assert.Equal(t, types.FileTypeImage, meta.Files[0].FileType)
	assert.Equal(t, "low", meta.Files[0].Detail)
}

func TestImageRequest_Unmarshal_Marshal_PreservesExtraFields(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-image-2",
		"prompt": "a cat",
		"n": 2,
		"size": "1024x1024",
		"quality": "high",
		"custom_field": "custom_value"
	}`)
	var req ImageRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	assert.Equal(t, "gpt-image-2", req.Model)
	assert.Equal(t, "a cat", req.Prompt)
	require.NotNil(t, req.N)
	assert.Equal(t, uint(2), *req.N)
	assert.Equal(t, "1024x1024", req.Size)
	assert.Equal(t, "high", req.Quality)
	assert.Equal(t, json.RawMessage(`"custom_value"`), req.Extra["custom_field"])

	encoded, err := common.Marshal(req)
	require.NoError(t, err)

	var restored ImageRequest
	require.NoError(t, common.Unmarshal(encoded, &restored))
	assert.Equal(t, json.RawMessage(`"custom_value"`), restored.Extra["custom_field"])
}
