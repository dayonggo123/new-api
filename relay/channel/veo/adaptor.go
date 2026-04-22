package veo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := strings.TrimSuffix(info.ChannelBaseUrl, "/")
	model := info.UpstreamModelName
	path, ok := modelToPath[model]
	if !ok {
		path = fmt.Sprintf("/uapi/v1/video-gen/%s", model)
	}
	return baseURL + path, nil
}

var modelToPath = map[string]string{
	// Video generation
	"veo-3.1":          "/uapi/v1/video-gen/veo",
	"veo-3.1-fast":     "/uapi/v1/video-gen/veo",
	"veo-2":            "/uapi/v1/video-gen/veo",
	"veo-3.1-lite":     "/uapi/v1/video-gen/veo",
	"grok-3":           "/uapi/v1/video-gen/grok",
	"grok-video":       "/uapi/v1/video-gen/grok",
	"seedance-2":       "/uapi/v1/video-gen/seedance",
	"seedance-2-remix": "/uapi/v1/video-gen/seedance",
	"seedance-2-omni":  "/uapi/v1/video-gen/seedance",
	"kling":            "/uapi/v1/video-gen/kling",
	// Image generation
	"nano-banana-pro":   "/uapi/v1/generate_image",
	"nano-banana-2":    "/uapi/v1/generate_image",
	"imagen-4":         "/uapi/v1/generate_image",
	"grok-image":       "/uapi/v1/imagen/grok",
	"meta-ai-image":    "/uapi/v1/meta_ai/generate",
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, header *http.Header, info *relaycommon.RelayInfo) error {
	header.Set("x-api-key", info.ApiKey)
	if info.ApiKey != "" {
		header.Set("Authorization", "Bearer "+info.ApiKey)
	}
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	prompt := ""
	// Get the LAST user message content (the actual prompt from user)
	for i := len(request.Messages) - 1; i >= 0; i-- {
		msg := request.Messages[i]
		if msg.Role == "user" {
			if content, ok := msg.Content.(string); ok && content != "" {
				prompt = content
				break
			}
		}
	}

	// Debug log what we extracted
	common.SysLog(fmt.Sprintf("veo ConvertOpenAIRequest: model=%s prompt=%q size=%v messages=%d",
		info.UpstreamModelName, prompt, request.Size, len(request.Messages)))

	buf, _, err := buildMultipartBodyFromFields(info.UpstreamModelName, prompt, request.Size)
	return buf, err
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	model := request.Model
	if model == "" {
		model = info.UpstreamModelName
	}
	buf, _, err := buildMultipartBodyFromFields(model, request.Prompt, request.Size)
	return buf, err
}

// imageModels returns true for image generation models (which use different resolution values)
var imageModels = map[string]bool{
	"nano-banana-pro": true,
	"nano-banana-2":   true,
	"imagen-4":        true,
	"grok-image":      true,
	"meta-ai-image":   true,
}

// buildMultipartBodyFromFields creates multipart form data from field values
func buildMultipartBodyFromFields(model, prompt, resolution string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("prompt", prompt)
	writer.WriteField("model", model)

	isImageModel := imageModels[model]
	switch {
	case resolution == "":
		// no resolution field
	case isImageModel:
		// Image models: resolution is "1K", "2K", "4K" or aspect_ratio like "16:9"
		// Only map if it looks like a video-style resolution
		switch {
		case strings.HasPrefix(resolution, "480"):
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "720"):
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "1080"):
			writer.WriteField("resolution", "2K")
		case strings.Contains(resolution, "x"):
			// "1024x1024" etc -> map to 1K
			writer.WriteField("resolution", "1K")
		case strings.HasPrefix(resolution, "1K") || strings.HasPrefix(resolution, "2K") || strings.HasPrefix(resolution, "4K"):
			writer.WriteField("resolution", resolution)
		// other values passed through as-is
		default:
			writer.WriteField("resolution", resolution)
		}
	case strings.HasPrefix(resolution, "480"):
		writer.WriteField("resolution", "480p")
	case strings.HasPrefix(resolution, "720"):
		writer.WriteField("resolution", "720p")
	case strings.HasPrefix(resolution, "1080"):
		writer.WriteField("resolution", "1080p")
	case strings.Contains(resolution, "x"):
		// e.g. "1024x1024" -> map to 720p
		writer.WriteField("resolution", "720p")
	case strings.HasPrefix(resolution, "square-"):
		writer.WriteField("resolution", "720p")
	// non-standard resolutions like "1024x1024" are dropped - upstream only accepts 480p/720p/1080p
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer failed: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	// requestBody is already the multipart body from ConvertOpenAIRequest
	// Just extract the content type and send it directly
	if buf, ok := requestBody.(*bytes.Buffer); ok && buf.Len() > 0 {
		contentType := "multipart/form-data; boundary=" + extractBoundaryFromMultipartBody(buf)
		c.Set("veo_multipart_content_type", contentType)
		return channel.DoFormRequestWithContentType(a, c, info, buf, contentType)
	}

	// Fallback: try to read from body storage
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	cachedBody, err := storage.Bytes()
	if err != nil {
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		// JSON request: parse and convert to multipart
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
			return nil, fmt.Errorf("unmarshal json body failed: %w", err)
		}

		model := info.UpstreamModelName
		if m, ok := bodyMap["model"].(string); ok && m != "" {
			model = m
		}

		prompt := ""
		if p, ok := bodyMap["prompt"].(string); ok {
			prompt = p
		}

		resolution := ""
		if r, ok := bodyMap["resolution"].(string); ok {
			resolution = r
		}

		multipartBody, multipartContentType, err := buildMultipartBodyFromFields(model, prompt, resolution)
		if err != nil {
			return nil, err
		}

		c.Set("veo_multipart_content_type", multipartContentType)
		return channel.DoFormRequestWithContentType(a, c, info, multipartBody, multipartContentType)
	} else if strings.Contains(contentType, "multipart/form-data") {
		// Already multipart: parse and rebuild with correct model
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, fmt.Errorf("parse multipart form failed: %w", err)
		}

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		model := info.UpstreamModelName
		if v, ok := formData.Value["model"]; ok && len(v) > 0 && v[0] != "" {
			model = v[0]
		}
		writer.WriteField("model", model)

		if v, ok := formData.Value["prompt"]; ok && len(v) > 0 {
			writer.WriteField("prompt", v[0])
		}
		if v, ok := formData.Value["resolution"]; ok && len(v) > 0 {
			writer.WriteField("resolution", v[0])
		}
		if v, ok := formData.Value["aspect_ratio"]; ok && len(v) > 0 {
			writer.WriteField("aspect_ratio", v[0])
		}
		if v, ok := formData.Value["mode"]; ok && len(v) > 0 {
			writer.WriteField("mode", v[0])
		}
		if v, ok := formData.Value["mode_image"]; ok && len(v) > 0 {
			writer.WriteField("mode_image", v[0])
		}
		if v, ok := formData.Value["duration"]; ok && len(v) > 0 {
			writer.WriteField("duration", v[0])
		}

		// Copy ref_images files
		for fieldName, fileHeaders := range formData.File {
			if fieldName != "ref_images" && fieldName != "files" && fieldName != "ref_videos" && fieldName != "ref_audios" {
				continue
			}
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}

		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close multipart writer failed: %w", err)
		}

		multipartBody := &buf
		multipartContentType := writer.FormDataContentType()

		c.Set("veo_multipart_content_type", multipartContentType)
		return channel.DoFormRequestWithContentType(a, c, info, multipartBody, multipartContentType)
	}

	// Unknown content type, use original
	return channel.DoApiRequest(a, c, info, requestBody)
}

// extractBoundaryFromMultipartBody extracts boundary from multipart body
func extractBoundaryFromMultipartBody(buf *bytes.Buffer) string {
	data := buf.String()
	eol := strings.Index(data, "\r\n")
	if eol == -1 {
		return ""
	}
	firstLine := data[:eol]
	const prefix = "--"
	if !strings.HasPrefix(firstLine, prefix) {
		return ""
	}
	return firstLine[len(prefix):]
}


func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if resp == nil {
		return nil, types.NewError(errors.New("empty response"), types.ErrorCodeBadResponse)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, types.NewError(readErr, types.ErrorCodeReadResponseBodyFailed)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, types.NewError(fmt.Errorf("bad status: %d body: %s", resp.StatusCode, string(body)), types.ErrorCodeBadResponse)
	}

	var submitResp submitResponse
	if err := common.Unmarshal(body, &submitResp); err != nil {
		return nil, types.NewError(fmt.Errorf("unmarshal failed: %w body: %s", err, string(body)), types.ErrorCodeBadResponseBody)
	}

	// Return OpenAI video response via c.JSON
	openaiResp := dto.NewOpenAIVideo()
	openaiResp.ID = submitResp.UUID
	openaiResp.Model = submitResp.ModelName
	openaiResp.CreatedAt = info.StartTime.Unix()

	c.JSON(http.StatusOK, openaiResp)

	usage = &dto.Usage{}
	return usage, nil
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

// submitResponse mirrors the upstream API response
type submitResponse struct {
	ID               int     `json:"id"`
	UUID             string  `json:"uuid"`
	UserID           int     `json:"user_id"`
	ModelName        string  `json:"model_name"`
	InputText        string  `json:"input_text"`
	Type             string  `json:"type"`
	Status           int     `json:"status"`
	StatusDesc       string  `json:"status_desc"`
	StatusPercentage int     `json:"status_percentage"`
	ErrorCode        string  `json:"error_code"`
	ErrorMessage     string  `json:"error_message"`
	EstimatedCredit  int     `json:"estimated_credit"`
	MediaType        string  `json:"media_type"`
	CreatedAt        string  `json:"created_at"`
	DelaySeconds     int     `json:"delay_seconds"`
}

// ModelList returns the supported model list
var ModelList = []string{
	// Video generation
	"veo-3.1",
	"veo-3.1-fast",
	"veo-2",
	"veo-3.1-lite",
	"grok-3",
	"grok-video",
	"seedance-2",
	"seedance-2-remix",
	"seedance-2-omni",
	"kling",
	// Image generation
	"nano-banana-pro",
	"nano-banana-2",
	"imagen-4",
	"grok-image",
	"meta-ai-image",
}

// ChannelName is the channel identifier
var ChannelName = "GeminiGen"
