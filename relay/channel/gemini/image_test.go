package gemini

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiImageGenerationHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns image b64_json", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respBody, err := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"index": 0,
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     "iVBORw0KGgoAAAA",
								},
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}

		info := &relaycommon.RelayInfo{}
		usage, apiErr := GeminiImageGenerationHandler(c, info, resp)
		require.Nil(t, apiErr)
		require.NotNil(t, usage)
		assert.Equal(t, 258, usage.PromptTokens)
		assert.Equal(t, 258, usage.TotalTokens)

		var imageResp dto.ImageResponse
		require.NoError(t, common.Unmarshal(w.Body.Bytes(), &imageResp))
		require.Len(t, imageResp.Data, 1)
		assert.Equal(t, "iVBORw0KGgoAAAA", imageResp.Data[0].B64Json)
	})

	t.Run("returns error when no images", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		respBody, err := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"index": 0,
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text": "only text",
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}

		info := &relaycommon.RelayInfo{}
		usage, apiErr := GeminiImageGenerationHandler(c, info, resp)
		require.Nil(t, usage)
		require.NotNil(t, apiErr)
	})
}
