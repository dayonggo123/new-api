package gemini

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// GeminiImageGenerationHandler parses a Gemini generateContent response that
// contains image inline data and returns an OpenAI-compatible image response.
// By default it returns b64_json.
func GeminiImageGenerationHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewOpenAIError(readErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	if common.DebugEnabled {
		println(string(responseBody))
	}

	var geminiResponse dto.GeminiImageGenerationResponse
	if err := common.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	openAIResponse := dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    make([]dto.ImageData, 0),
	}

	var imageCount int
	for _, candidate := range geminiResponse.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil || part.InlineData.Data == "" {
				continue
			}
			if !strings.HasPrefix(part.InlineData.MimeType, "image") {
				continue
			}
			imageCount++
			openAIResponse.Data = append(openAIResponse.Data, dto.ImageData{
				B64Json: part.InlineData.Data,
			})
		}
	}

	if len(openAIResponse.Data) == 0 {
		return nil, types.NewOpenAIError(
			errors.New("no images generated"),
			types.ErrorCodeBadResponseBody,
			http.StatusInternalServerError,
		)
	}

	jsonResponse, marshalErr := common.Marshal(openAIResponse)
	if marshalErr != nil {
		return nil, types.NewError(marshalErr, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)

	// Each generated image is billed at a fixed 258 tokens, consistent with the
	// existing imagen handler.
	const imageTokens = 258
	usage := &dto.Usage{
		PromptTokens:     imageTokens * imageCount,
		CompletionTokens: 0,
		TotalTokens:      imageTokens * imageCount,
	}
	return usage, nil
}

// decodeBase64Image decodes a base64 string and returns the raw bytes. It
// accepts both raw base64 and data URI formats.
func decodeBase64Image(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "data:") {
		_, base64String, err := service.DecodeBase64FileData(data)
		if err != nil {
			return nil, err
		}
		data = base64String
	}
	return base64.StdEncoding.DecodeString(data)
}
