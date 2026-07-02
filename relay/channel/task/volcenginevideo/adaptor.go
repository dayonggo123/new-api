package volcenginevideo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// TaskAdaptor implements the TaskAdaptor interface for VolcEngine Ark video
// generation (POST /api/v3/contents/generations/tasks).
type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

// Init loads channel metadata from the relay info.
func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses the incoming request, validates required
// fields and stores the parsed request in the Gin context for later use.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL returns the VolcEngine Ark content generation endpoint.
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v3/contents/generations/tasks", a.baseURL), nil
}

// BuildRequestHeader sets the JSON and Bearer authorization headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// BuildRequestBody converts the OpenAI-compatible task request into a VolcEngine
// video generation payload.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body := a.convertToRequestPayload(&req)
	if info.IsModelMapped {
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request body failed")
	}
	return bytes.NewReader(data), nil
}

// DoRequest delegates to the shared task API request helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse parses the upstream submit response, writes an OpenAI-compatible
// queued task object to the client, and returns the upstream task ID for
// persistence.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var submitResp VolcengineVideoSubmitResponse
	if err := common.Unmarshal(responseBody, &submitResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if submitResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return submitResp.ID, responseBody, nil
}

// FetchTask queries the upstream task status.
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/v3/contents/generations/tasks/%s", baseUrl, taskID)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

// ParseTaskResult maps the upstream VolcEngine status to internal task status.
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var result VolcengineVideoTaskResult
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskInfo := relaycommon.TaskInfo{
		Code: 0,
	}

	switch result.Status {
	case "queued", "pending":
		taskInfo.Status = model.TaskStatusQueued
		taskInfo.Progress = "10%"
	case "running", "processing":
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "50%"
	case "succeeded":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		taskInfo.Url = result.Content.VideoURL
		taskInfo.CompletionTokens = result.Usage.CompletionTokens
		taskInfo.TotalTokens = result.Usage.TotalTokens
	case "failed", "expired", "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = result.Error.Message
	default:
		// Unknown status: assume still in progress and wait for the next poll.
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "30%"
	}

	return &taskInfo, nil
}

// EstimateBilling returns OtherRatios used for pre-charging based on duration,
// resolution and video-reference discounts.
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	ratios := make(map[string]float64)

	duration := req.Duration
	if duration <= 0 {
		if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
			duration = sec
		}
	}
	if duration > 0 {
		ratios["duration"] = taskcommon.EstimateDurationRatio(duration)
	}

	resolution := req.Resolution
	if resolution == "" {
		resolution = "720p"
	}
	if resRatio := taskcommon.EstimateResolutionRatio(resolution); resRatio != 1.0 {
		ratios["resolution"] = resRatio
	}

	if taskcommon.HasVideoReference(req) {
		if ratio, ok := GetVideoInputRatio(info.OriginModelName); ok {
			ratios["video_input"] = ratio
		}
	}

	if len(ratios) == 0 {
		return nil
	}
	return ratios
}

// GetModelList returns the supported model list for the VolcEngine video channel.
func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName returns the channel display name.
func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// convertToRequestPayload builds the VolcEngine content-generation request from
// the OpenAI-compatible task submit request.
func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) *VolcengineVideoRequest {
	payload := &VolcengineVideoRequest{
		Model:   req.Model,
		Content: []VolcengineContentItem{},
	}

	// Prompt: type=text, role=user.
	payload.Content = append(payload.Content, VolcengineContentItem{
		Type: "text",
		Role: "user",
		Text: req.Prompt,
	})

	// Images: first image -> first_frame, remaining -> reference_image.
	images := req.Images
	if len(images) == 0 && req.Image != "" {
		images = []string{req.Image}
	}
	images = append(images, req.ImageURLs...)
	for i, imgURL := range images {
		role := "first_frame"
		if i > 0 {
			role = "reference_image"
		}
		payload.Content = append(payload.Content, VolcengineContentItem{
			Type:     "image_url",
			Role:     role,
			ImageURL: &MediaURL{URL: imgURL},
		})
	}

	// Reference images: role=reference_image.
	for _, imgURL := range req.ReferenceImages {
		payload.Content = append(payload.Content, VolcengineContentItem{
			Type:     "image_url",
			Role:     "reference_image",
			ImageURL: &MediaURL{URL: imgURL},
		})
	}

	// Reference video: prefer reference_video, fallback to legacy video_urls.
	referenceVideos := req.ReferenceVideo
	if len(referenceVideos) == 0 {
		referenceVideos = req.VideoURLs
	}
	for _, videoURL := range referenceVideos {
		payload.Content = append(payload.Content, VolcengineContentItem{
			Type:     "video_url",
			Role:     "reference_video",
			VideoURL: &MediaURL{URL: videoURL},
		})
	}

	// Reference audio: role=reference_audio.
	for _, audioURL := range req.ReferenceAudio {
		payload.Content = append(payload.Content, VolcengineContentItem{
			Type:     "audio_url",
			Role:     "reference_audio",
			AudioURL: &MediaURL{URL: audioURL},
		})
	}

	// Ratio mapping: explicit ratio takes precedence, then size, then aspect_ratio.
	if req.Ratio != "" {
		payload.Ratio = req.Ratio
	} else if req.Size != "" {
		ratio, _ := taskcommon.ParseSizeToRatio(req.Size)
		payload.Ratio = ratio
	} else if req.AspectRatio != "" {
		payload.Ratio = req.AspectRatio
	}

	// Duration: default 5 seconds.
	duration := req.Duration
	if duration <= 0 {
		if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
			duration = sec
		}
	}
	if duration <= 0 {
		duration = 5
	}
	payload.Duration = duration

	// Resolution: default 720p.
	if req.Resolution != "" {
		payload.Resolution = req.Resolution
	} else {
		payload.Resolution = "720p"
	}

	// generate_audio defaults to true; watermark defaults to false.
	if req.GenerateAudio != nil {
		payload.GenerateAudio = *req.GenerateAudio
	} else {
		payload.GenerateAudio = true
	}
	if req.Watermark != nil {
		payload.Watermark = *req.Watermark
	} else {
		payload.Watermark = false
	}

	if req.Seed > 0 {
		payload.Seed = req.Seed
	}
	if req.CameraFixed != nil {
		payload.CameraFixed = *req.CameraFixed
	}
	if req.Frames > 0 {
		payload.Frames = req.Frames
	}
	if req.Priority != "" {
		payload.Priority = req.Priority
	}
	if req.ServiceTier != "" {
		payload.ServiceTier = req.ServiceTier
	}
	if req.CallbackURL != "" {
		payload.CallbackURL = req.CallbackURL
	}

	return payload
}
