package omniflash

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
)

// TaskAdaptor implements the Omni Flash Interactions API task adapter.
// It supports text-to-video, image-to-video and reference video inputs.
type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelName := info.UpstreamModelName
	version := model_setting.GetGeminiVersionSetting(modelName)
	return fmt.Sprintf("%s/%s/models/%s:interactions", a.baseURL, version, modelName), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, ok := c.Get("task_request")
	if !ok {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("unexpected task_request type")
	}

	omniReq := OmniFlashRequest{
		Input: []OmniFlashInput{
			{Text: req.Prompt},
		},
	}

	// Main image input.
	if len(req.Images) > 0 {
		if img := parseImageInput(req.Images[0]); img != nil {
			omniReq.Input = append(omniReq.Input, OmniFlashInput{Image: img})
			info.Action = constant.TaskActionGenerate
		}
	} else if req.Image != "" {
		if img := parseImageInput(req.Image); img != nil {
			omniReq.Input = append(omniReq.Input, OmniFlashInput{Image: img})
			info.Action = constant.TaskActionGenerate
		}
	}

	// Reference video input.
	if len(req.VideoURLs) > 0 {
		omniReq.Input = append(omniReq.Input, OmniFlashInput{
			Reference: &OmniFlashReference{Video: &OmniFlashVideo{URI: req.VideoURLs[0]}},
		})
	} else if len(req.ReferenceVideo) > 0 {
		omniReq.Input = append(omniReq.Input, OmniFlashInput{
			Reference: &OmniFlashReference{Video: &OmniFlashVideo{URI: req.ReferenceVideo[0]}},
		})
	}

	// Config from metadata / standard fields.
	config := OmniFlashConfig{
		AspectRatio:     req.AspectRatio,
		Resolution:      req.Resolution,
		DurationSeconds: req.Duration,
		SessionID:       getMetadataString(req.Metadata, "session_id"),
	}
	if config.AspectRatio == "" && req.Size != "" {
		config.AspectRatio = SizeToOmniAspectRatio(req.Size)
	}
	if config.Resolution == "" && req.Size != "" {
		config.Resolution = SizeToOmniResolution(req.Size)
	}
	if config.DurationSeconds == 0 {
		if seconds := common.String2Int(req.Seconds); seconds > 0 {
			config.DurationSeconds = seconds
		}
	}
	if config.DurationSeconds == 0 {
		config.DurationSeconds = getMetadataInt(req.Metadata, "duration_seconds")
	}
	if config.Resolution == "" {
		config.Resolution = getMetadataString(req.Metadata, "resolution")
	}
	if config.AspectRatio == "" {
		config.AspectRatio = getMetadataString(req.Metadata, "aspect_ratio")
	}
	omniReq.Config = config

	data, err := common.Marshal(omniReq)
	if err != nil {
		return nil, fmt.Errorf("marshal omni flash request failed: %w", err)
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var s OmniFlashSubmitResponse
	if err := common.Unmarshal(responseBody, &s); err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
	}
	if strings.TrimSpace(s.Name) == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("missing interaction name"), "invalid_response", http.StatusInternalServerError)
	}
	taskID = taskcommon.EncodeLocalTaskID(s.Name)
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return taskID, responseBody, nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	v, ok := c.Get("task_request")
	if !ok {
		return nil
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil
	}
	seconds := req.Duration
	if seconds == 0 {
		seconds = common.String2Int(req.Seconds)
	}
	if seconds == 0 {
		seconds = getMetadataInt(req.Metadata, "duration_seconds")
	}
	if seconds == 0 {
		seconds = 4
	}
	resolution := ResolveOmniResolution(req.Metadata, req.Resolution, req.Size)
	resRatio := OmniResolutionRatio(resolution)
	return map[string]float64{
		"seconds":    float64(seconds),
		"resolution": resRatio,
	}
}

func (a *TaskAdaptor) AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	return 0
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	upstreamName, err := taskcommon.DecodeLocalTaskID(taskID)
	if err != nil {
		return nil, fmt.Errorf("decode task_id failed: %w", err)
	}
	version := model_setting.GetGeminiVersionSetting("default")
	url := fmt.Sprintf("%s/%s/%s", baseUrl, version, upstreamName)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-goog-api-key", key)
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var op OmniFlashOperationResponse
	if err := common.Unmarshal(respBody, &op); err != nil {
		return nil, fmt.Errorf("unmarshal operation response failed: %w", err)
	}
	ti := &relaycommon.TaskInfo{}
	if op.Error.Message != "" {
		ti.Status = model.TaskStatusFailure
		ti.Reason = op.Error.Message
		ti.Progress = "100%"
		return ti, nil
	}
	if !op.Done {
		ti.Status = model.TaskStatusInProgress
		ti.Progress = "50%"
		return ti, nil
	}
	ti.Status = model.TaskStatusSuccess
	ti.Progress = "100%"
	ti.TaskID = taskcommon.EncodeLocalTaskID(op.Name)
	if len(op.Response.GeneratedVideos) > 0 {
		if uri := op.Response.GeneratedVideos[0].Video.URI; uri != "" {
			ti.RemoteUrl = uri
		}
	}
	return ti, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	video := dto.NewOpenAIVideo()
	video.ID = task.TaskID
	video.Model = task.Properties.OriginModelName
	if video.Model == "" {
		video.Model = task.Properties.UpstreamModelName
	}
	video.Status = task.Status.ToVideoStatus()
	video.SetProgressStr(task.Progress)
	video.CreatedAt = task.CreatedAt
	if task.FinishTime > 0 {
		video.CompletedAt = task.FinishTime
	} else if task.UpdatedAt > 0 {
		video.CompletedAt = task.UpdatedAt
	}
	url := task.GetResultURL()
	if url == "" {
		url = taskcommon.BuildProxyURL(task.TaskID)
	}
	if url == "" {
		url = task.PrivateData.ResultURL
	}
	if url != "" {
		video.SetMetadata("url", url)
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"gemini-omni-flash-preview",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "gemini"
}

// ============================
// helpers
// ============================

func parseImageInput(imageStr string) *OmniFlashImage {
	imageStr = strings.TrimSpace(imageStr)
	if imageStr == "" {
		return nil
	}
	if strings.HasPrefix(imageStr, "data:") {
		rest := imageStr[len("data:"):]
		idx := strings.Index(rest, ",")
		if idx < 0 {
			return nil
		}
		meta := rest[:idx]
		b64 := rest[idx+1:]
		if b64 == "" {
			return nil
		}
		mimeType := "application/octet-stream"
		parts := strings.SplitN(meta, ";", 2)
		if len(parts) >= 1 && parts[0] != "" {
			mimeType = parts[0]
		}
		return &OmniFlashImage{
			BytesBase64Encoded: b64,
			MimeType:           mimeType,
		}
	}
	raw, err := base64.StdEncoding.DecodeString(imageStr)
	if err != nil {
		return nil
	}
	return &OmniFlashImage{
		BytesBase64Encoded: imageStr,
		MimeType:           http.DetectContentType(raw),
	}
}

func SizeToOmniAspectRatio(size string) string {
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return "16:9"
	}
	w := common.String2Int(parts[0])
	h := common.String2Int(parts[1])
	if w <= 0 || h <= 0 {
		return "16:9"
	}
	if h > w {
		return "9:16"
	}
	return "16:9"
}

func SizeToOmniResolution(size string) string {
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return "720p"
	}
	w := common.String2Int(parts[0])
	h := common.String2Int(parts[1])
	maxDim := w
	if h > maxDim {
		maxDim = h
	}
	if maxDim >= 3840 {
		return "4k"
	}
	if maxDim >= 1920 {
		return "1080p"
	}
	return "720p"
}

func ResolveOmniResolution(metadata map[string]any, resolution, size string) string {
	if resolution != "" {
		return strings.ToLower(resolution)
	}
	if metadata != nil {
		if v, ok := metadata["resolution"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return strings.ToLower(s)
			}
		}
	}
	if size != "" {
		return SizeToOmniResolution(size)
	}
	return "720p"
}

func OmniResolutionRatio(resolution string) float64 {
	if resolution == "4k" {
		return 1.5
	}
	return 1.0
}

func getMetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	v, ok := metadata[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getMetadataInt(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	v, ok := metadata[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}
