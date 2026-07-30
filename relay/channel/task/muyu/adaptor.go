package muyu

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// TaskAdaptor implements the TaskAdaptor interface for Muyu video generation
type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

// Init loads channel metadata from the relay info
func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses the incoming request and validates required fields
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL returns the unified task creation endpoint
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s%s", a.baseURL, TasksEndpoint), nil
}

// BuildRequestHeader sets request headers
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return nil
}

// BuildRequestBody converts the OpenAI-compatible task request into Muyu task request
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body := a.convertToRequestPayload(&req, info)
	if info.IsModelMapped {
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
	}

	// Add API key to request body
	body.API = a.apiKey

	data, err := common.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request body failed")
	}

	return bytes.NewReader(data), nil
}

// DoRequest delegates to the shared task API request helper
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse parses the upstream submit response, writes OpenAI-compatible response
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var createResp TaskCreateResponse
	if err := common.Unmarshal(responseBody, &createResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if !createResp.Success {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("muyu api error: %s", createResp.Message), "create_failed", http.StatusBadRequest)
		return
	}

	if createResp.TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return createResp.TaskID, responseBody, nil
}

// FetchTask queries the upstream task status
func (a *TaskAdaptor) FetchTask(baseUrl, apiKey string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s%s/%s?api=%s", baseUrl, TasksEndpoint, taskID, apiKey)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

// ParseTaskResult maps the upstream Muyu status to internal task status
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var result TaskQueryResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskInfo := relaycommon.TaskInfo{
		Code: 0,
	}

	if !result.Success {
		taskInfo.Code = -1
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = result.Message
		return &taskInfo, nil
	}

	switch result.Status {
	case StatusGenerating, StatusSubmitting, StatusSubmissionUnknown:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "30%"
	case StatusSuccess:
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		taskInfo.Url = result.ResultURL
	case StatusFailed:
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = result.Message
	default:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = "30%"
	}

	return &taskInfo, nil
}

// EstimateBilling returns ratios used for pre-charging based on duration and resolution
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	ratios := make(map[string]float64)

	// Duration ratio
	duration := req.Duration
	if duration <= 0 {
		if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
			duration = sec
		}
	}
	if duration > 0 {
		ratios["duration"] = taskcommon.EstimateDurationRatio(duration)
	}

	// Resolution ratio
	resolution := req.Resolution
	if resolution == "" {
		resolution = DefaultResolution
	}
	if resRatio := taskcommon.EstimateResolutionRatio(resolution); resRatio != 1.0 {
		ratios["resolution"] = resRatio
	}

	// Video reference ratio
	if taskcommon.HasVideoReference(req) {
		ratios["video_input"] = 1.5 // typical video reference multiplier
	}

	// Audio reference ratio
	if len(req.ReferenceAudio) > 0 {
		ratios["audio_input"] = 1.2
	}

	if len(ratios) == 0 {
		return nil
	}
	return ratios
}

// GetModelList returns the supported model list for Muyu video channel
func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// GetChannelName returns the channel display name
func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// convertToRequestPayload builds the Muyu unified task request from OpenAI-compatible request
func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) *TaskCreateRequest {
	payload := &TaskCreateRequest{
		Model:      req.Model,
		Prompt:     req.Prompt,
		Parameters: make(map[string]interface{}),
	}

	// Collect asset IDs from various sources
	assetIDs := a.collectAssetIDs(req)
	if len(assetIDs) > 0 {
		payload.AssetIDs = assetIDs
	}

	// Build parameters object
	params := payload.Parameters

	// Aspect ratio
	if req.Ratio != "" {
		params["aspect"] = a.normalizeAspect(req.Ratio)
	} else if req.Size != "" {
		ratio, _ := taskcommon.ParseSizeToRatio(req.Size)
		params["aspect"] = ratio
	} else if req.AspectRatio != "" {
		params["aspect"] = a.normalizeAspect(req.AspectRatio)
	} else {
		params["aspect"] = DefaultAspect
	}

	// Duration
	duration := req.Duration
	if duration <= 0 {
		if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
			duration = sec
		}
	}
	if duration <= 0 {
		duration = DefaultDuration
	}
	params["duration"] = duration

	// Resolution (for channel6 models)
	if strings.HasPrefix(req.Model, "channel6/") {
		if req.Resolution != "" {
			params["resolution"] = a.normalizeResolution(req.Resolution)
		} else {
			params["resolution"] = DefaultResolution
		}

		// Generate audio
		if req.GenerateAudio != nil {
			params["generateAudio"] = *req.GenerateAudio
		} else {
			params["generateAudio"] = true
		}
	}

	return payload
}

// collectAssetIDs gathers asset IDs from various request fields
func (a *TaskAdaptor) collectAssetIDs(req *relaycommon.TaskSubmitReq) []string {
	var assetIDs []string

	// From images - check if already an asset ID (starts with "ast_")
	for _, img := range req.Images {
		if img != "" && strings.HasPrefix(img, "ast_") {
			assetIDs = append(assetIDs, img)
		}
	}

	// From image URLs
	for _, imgURL := range req.ImageURLs {
		if imgURL != "" && strings.HasPrefix(imgURL, "ast_") {
			assetIDs = append(assetIDs, imgURL)
		}
	}

	// From reference images
	for _, img := range req.ReferenceImages {
		if img != "" && strings.HasPrefix(img, "ast_") {
			assetIDs = append(assetIDs, img)
		}
	}

	return assetIDs
}

// normalizeAspect converts various aspect ratio formats to Muyu format
func (a *TaskAdaptor) normalizeAspect(ratio string) string {
	switch ratio {
	case "16:9", "16/9":
		return "16:9"
	case "9:16", "9/16":
		return "9:16"
	case "1:1", "1/1":
		return "1:1"
	case "4:3", "4/3":
		return "4:3"
	case "3:4", "3/4":
		return "3:4"
	default:
		return DefaultAspect
	}
}

// normalizeResolution converts resolution to Muyu format
func (a *TaskAdaptor) normalizeResolution(res string) string {
	switch {
	case strings.Contains(res, "1080"):
		return "1080p"
	case strings.Contains(res, "720"):
		return "720p"
	case strings.Contains(res, "480"):
		return "480p"
	default:
		return DefaultResolution
	}
}

// ConvertToOpenAIVideo converts stored task data to OpenAI video format
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var muyuResp TaskQueryResponse
	if err := common.Unmarshal(originTask.Data, &muyuResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal muyu task data failed")
	}

	openAIVideo := originTask.ToOpenAIVideo()
	if !muyuResp.Success {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: muyuResp.Message,
			Code:    "1",
		}
	}

	jsonData, err := common.Marshal(openAIVideo)
	if err != nil {
		return nil, errors.Wrap(err, "marshal openai video failed")
	}

	return jsonData, nil
}
