package volcengine

import (
	"encoding/json"
	"errors"
	"fmt"
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

// VolcengineImageRequest mirrors the VolcEngine Ark /api/v3/images/generations request body.
type VolcengineImageRequest struct {
	Model                            string                     `json:"model"`
	Prompt                           string                     `json:"prompt"`
	N                                *uint                      `json:"n,omitempty"`
	Size                             string                     `json:"size,omitempty"`
	Quality                          string                     `json:"quality,omitempty"`
	ResponseFormat                   string                     `json:"response_format,omitempty"`
	Style                            json.RawMessage            `json:"style,omitempty"`
	OutputFormat                     json.RawMessage            `json:"output_format,omitempty"`
	Watermark                        *bool                      `json:"watermark,omitempty"`
	Image                            any                        `json:"image,omitempty"` // string or []string
	SequentialImageGeneration        string                     `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions json.RawMessage            `json:"sequential_image_generation_options,omitempty"`
	Extra                            map[string]json.RawMessage `json:"-"`
}

// VolcengineImageResponse is the raw response from VolcEngine image generation API.
type VolcengineImageResponse struct {
	Created int64                  `json:"created"`
	Data    []VolcengineImageData  `json:"data"`
}

type VolcengineImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// oaiImage2VolcengineImageRequest converts an OpenAI-compatible ImageRequest into the
// VolcEngine Ark image generation request format.
func oaiImage2VolcengineImageRequest(request *dto.ImageRequest) (*VolcengineImageRequest, error) {
	if request == nil {
		return nil, errors.New("image request is nil")
	}

	volcRequest := &VolcengineImageRequest{
		Model:          request.Model,
		Prompt:         request.Prompt,
		N:              request.N,
		Size:           request.Size,
		Quality:        request.Quality,
		ResponseFormat: request.ResponseFormat,
		Style:          request.Style,
		Watermark:      request.Watermark,
		Extra:          request.Extra,
	}

	if request.OutputFormat != nil {
		volcRequest.OutputFormat = request.OutputFormat
	}

	// image may be passed as a single URL or as an array of reference URLs.
	if request.Image != nil {
		var imageStr string
		if err := json.Unmarshal(request.Image, &imageStr); err == nil && imageStr != "" {
			volcRequest.Image = imageStr
		} else {
			var imageArray []string
			if err := json.Unmarshal(request.Image, &imageArray); err == nil && len(imageArray) > 0 {
				volcRequest.Image = imageArray
			}
		}
	}

	// Extra fields that the user may pass directly under the top-level request.
	if raw, ok := request.Extra["sequential_image_generation"]; ok {
		var seq string
		if err := common.Unmarshal(raw, &seq); err == nil {
			volcRequest.SequentialImageGeneration = seq
		}
		delete(volcRequest.Extra, "sequential_image_generation")
	}
	if raw, ok := request.Extra["sequential_image_generation_options"]; ok {
		volcRequest.SequentialImageGenerationOptions = raw
		delete(volcRequest.Extra, "sequential_image_generation_options")
	}
	if raw, ok := request.Extra["output_format"]; ok {
		volcRequest.OutputFormat = raw
		delete(volcRequest.Extra, "output_format")
	}

	return volcRequest, nil
}

// volcengineImageRequestToMap serializes the request to a map so that Extra fields can be merged.
func volcengineImageRequestToMap(req *VolcengineImageRequest) (map[string]any, error) {
	data, err := common.Marshal(req)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := common.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for k, v := range req.Extra {
		if _, exists := m[k]; !exists {
			var val any
			if err := common.Unmarshal(v, &val); err == nil {
				m[k] = val
			}
		}
	}
	return m, nil
}

// volcengineImageResponse2OpenAI converts a VolcEngine image response into the OpenAI-compatible
// ImageResponse. If the client requested b64_json, remote URLs are downloaded and encoded.
func volcengineImageResponse2OpenAI(c *gin.Context, volcResp *VolcengineImageResponse, request *dto.ImageRequest, info *relaycommon.RelayInfo) (*dto.ImageResponse, error) {
	if volcResp == nil {
		return nil, errors.New("empty volcengine image response")
	}

	openAIResp := &dto.ImageResponse{
		Created: info.StartTime.Unix(),
	}

	wantsBase64 := strings.EqualFold(request.ResponseFormat, "b64_json")

	for _, item := range volcResp.Data {
		data := dto.ImageData{
			RevisedPrompt: item.RevisedPrompt,
		}
		if wantsBase64 && item.URL != "" {
			b64, err := downloadImageToBase64(c, item.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to download image for b64_json: %w", err)
			}
			data.B64Json = b64
		} else {
			data.Url = item.URL
			data.B64Json = item.B64JSON
		}
		openAIResp.Data = append(openAIResp.Data, data)
	}

	if len(openAIResp.Data) == 0 {
		return nil, errors.New("volcengine image response contains no image data")
	}

	return openAIResp, nil
}

// downloadImageToBase64 downloads an image URL and returns it as a data URL.
func downloadImageToBase64(c *gin.Context, url string) (string, error) {
	if url == "" {
		return "", errors.New("image url is empty")
	}
	_, data, err := service.GetImageFromUrl(url)
	if err != nil {
		return "", fmt.Errorf("failed to download image from %s: %w", url, err)
	}
	return data, nil
}

// volcengineImageHandler reads the upstream HTTP response and writes an OpenAI-compatible JSON body.
func volcengineImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	var volcResp VolcengineImageResponse
	if err := common.Unmarshal(responseBody, &volcResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	imageReq, _ := info.Request.(*dto.ImageRequest)
	openAIResp, err := volcengineImageResponse2OpenAI(c, &volcResp, imageReq, info)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	jsonResponse, err := common.Marshal(openAIResp)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err := c.Writer.Write(jsonResponse); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	return &dto.Usage{}, nil
}
