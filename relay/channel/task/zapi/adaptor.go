package zapi

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

// TaskAdaptor implements the task-based channel for Z-api (Seedance 2.0-fast).
// It submits asynchronous video generation tasks to Z-api and polls for results
// in the OpenAI Video API compatible format.
type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL string
	apiKey  string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/tasks", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"model":  info.UpstreamModelName,
		"prompt": req.Prompt,
	}

	// duration: prefer TaskSubmitReq.Duration, then metadata["duration"], default 10
	duration := req.Duration
	if duration == 0 && req.Metadata != nil {
		if raw, ok := req.Metadata["duration"]; ok {
			duration = parseInt(raw)
		}
	}
	if duration == 0 {
		duration = 10
	}
	payload["duration"] = duration

	// ratio: prefer TaskSubmitReq.Ratio, then AspectRatio, then Size, then metadata["ratio"], default 16:9
	ratio := req.Ratio
	if ratio == "" && req.AspectRatio != "" {
		ratio = req.AspectRatio
	}
	if ratio == "" && req.Size != "" {
		if strings.Contains(req.Size, ":") {
			ratio = req.Size
		} else {
			ratio, _ = taskcommon.ParseSizeToRatio(req.Size)
		}
	}
	if ratio == "" && req.Metadata != nil {
		if raw, ok := req.Metadata["ratio"]; ok {
			if s, ok := raw.(string); ok {
				ratio = s
			}
		}
	}
	if ratio == "" {
		ratio = "16:9"
	}
	payload["ratio"] = ratio

	// resolution: prefer TaskSubmitReq.Resolution, then metadata["resolution"], default 720p
	resolution := req.Resolution
	if resolution == "" && req.Metadata != nil {
		if raw, ok := req.Metadata["resolution"]; ok {
			if s, ok := raw.(string); ok {
				resolution = s
			}
		}
	}
	if resolution == "" {
		resolution = "720p"
	}
	payload["resolution"] = resolution

	// reference images: normalize from ImageURLs/Images/ReferenceImages/Image
	var images []string
	if len(req.ReferenceImages) > 0 {
		images = req.ReferenceImages
	} else if len(req.Images) > 0 {
		images = req.Images
	} else if len(req.ImageURLs) > 0 {
		images = req.ImageURLs
	} else if req.Image != "" {
		images = []string{req.Image}
	}
	if len(images) > 0 {
		payload["referenceImages"] = images
	}

	// first / last frame images (used for start/end frame video generation)
	if req.Metadata != nil {
		if raw, ok := req.Metadata["first_image"]; ok {
			if s, ok := raw.(string); ok && s != "" {
				payload["first_image"] = s
			}
		}
		if raw, ok := req.Metadata["last_image"]; ok {
			if s, ok := raw.(string); ok && s != "" {
				payload["last_image"] = s
			}
		}
	}

	// reference videos
	var videos []string
	if len(req.VideoURLs) > 0 {
		videos = req.VideoURLs
	} else if len(req.ReferenceVideo) > 0 {
		videos = req.ReferenceVideo
	}
	if len(videos) > 0 {
		payload["referenceVideos"] = videos
	}

	// reference audios
	var audios []string
	if len(req.ReferenceAudio) > 0 {
		audios = req.ReferenceAudio
	}
	if len(audios) > 0 {
		payload["referenceAudios"] = audios
	}

	// Merge remaining metadata fields that are not already handled.
	knownFields := map[string]bool{
		"model": true, "prompt": true, "duration": true, "ratio": true,
		"resolution": true, "first_image": true, "last_image": true,
		"referenceImages": true, "referenceVideos": true, "referenceAudios": true,
		"images": true, "videos": true, "audios": true, "image": true,
		"size": true, "aspect_ratio": true,
	}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			if knownFields[k] {
				continue
			}
			if _, exists := payload[k]; !exists {
				payload[k] = v
			}
		}
	}

	body, err := common.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal zapi request body failed")
	}
	common.SysLog(fmt.Sprintf("[ZAPI] request body: %s", string(body)))
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// submitResponse matches Z-api's POST /v1/tasks response.
type submitResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// queryResponse matches Z-api's GET /v1/tasks/{task_id} response.
type queryResponse struct {
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	ResultURL      string `json:"result_url,omitempty"`
	VideoURL       string `json:"video_url,omitempty"`
	ActualDuration string `json:"actualDuration,omitempty"`
	Amount         string `json:"amount,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	if resp == nil || resp.Body == nil {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("response or response body is nil"), "nil_response", http.StatusInternalServerError)
		return
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var sResp submitResponse
	if err := common.Unmarshal(responseBody, &sResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	common.SysLog(fmt.Sprintf("[ZAPI] submit response body: %s", string(responseBody)))

	if sResp.TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// Return OpenAI-compatible response to downstream client.
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return sResp.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	url := fmt.Sprintf("%s/v1/tasks/%s", baseUrl, taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var qResp queryResponse
	if err := common.Unmarshal(respBody, &qResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal query response failed")
	}

	taskInfo := &relaycommon.TaskInfo{
		Code: 0,
	}

	status := strings.ToUpper(qResp.Status)
	switch status {
	case "SUCCESS":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if qResp.ResultURL != "" {
			taskInfo.Url = qResp.ResultURL
		} else if qResp.VideoURL != "" {
			taskInfo.Url = qResp.VideoURL
		}
	case "FAILURE", "FAILED":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = qResp.FailureReason
		if taskInfo.Reason == "" {
			taskInfo.Reason = qResp.Status
		}
	case "QUEUED", "IN_PROGRESS":
		taskInfo.Status = model.TaskStatusInProgress
	default:
		taskInfo.Status = model.TaskStatusInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	openAIVideo.CreatedAt = task.CreatedAt
	openAIVideo.CompletedAt = task.UpdatedAt

	if task.PrivateData.ResultURL != "" {
		openAIVideo.SetMetadata("url", task.PrivateData.ResultURL)
	}

	if task.Status == model.TaskStatusFailure && task.FailReason != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: task.FailReason,
			Code:    "task_failed",
		}
	}

	return common.Marshal(openAIVideo)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"seedance-2.0-fast(431)",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "zapi"
}

func parseInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}
