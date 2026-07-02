package dto

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageRequest_NewFieldsPassThrough(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-image-2",
		"prompt": "a cat",
		"style": "vivid",
		"background": "transparent",
		"output_format": "png",
		"output_compression": 80,
		"image": "https://example.com/ref.png",
		"user": "user-123",
		"n": 1,
		"size": "1024x1024"
	}`)

	var req ImageRequest
	require.NoError(t, common.Unmarshal(raw, &req))

	assert.Equal(t, "gpt-image-2", req.Model)
	assert.Equal(t, "a cat", req.Prompt)
	assert.Equal(t, json.RawMessage(`"vivid"`), req.Style)
	assert.Equal(t, json.RawMessage(`"transparent"`), req.Background)
	assert.Equal(t, json.RawMessage(`"png"`), req.OutputFormat)
	assert.Equal(t, json.RawMessage(`80`), req.OutputCompression)
	assert.Equal(t, json.RawMessage(`"https://example.com/ref.png"`), req.Image)
	assert.Equal(t, json.RawMessage(`"user-123"`), req.User)

	encoded, err := common.Marshal(req)
	require.NoError(t, err)

	var restored ImageRequest
	require.NoError(t, common.Unmarshal(encoded, &restored))
	assert.Equal(t, json.RawMessage(`"vivid"`), restored.Style)
	assert.Equal(t, json.RawMessage(`"transparent"`), restored.Background)
	assert.Equal(t, json.RawMessage(`"png"`), restored.OutputFormat)
	assert.Equal(t, json.RawMessage(`80`), restored.OutputCompression)
	assert.Equal(t, json.RawMessage(`"https://example.com/ref.png"`), restored.Image)
	assert.Equal(t, json.RawMessage(`"user-123"`), restored.User)
}

func TestImageRequest_NewFieldsArrayImage(t *testing.T) {
	raw := []byte(`{
		"model": "gpt-image-2",
		"prompt": "a cat",
		"image": ["https://example.com/1.png", "https://example.com/2.png"]
	}`)

	var req ImageRequest
	require.NoError(t, common.Unmarshal(raw, &req))
	require.NotNil(t, req.Image)

	var urls []string
	require.NoError(t, common.Unmarshal(req.Image, &urls))
	assert.Equal(t, []string{"https://example.com/1.png", "https://example.com/2.png"}, urls)
}
