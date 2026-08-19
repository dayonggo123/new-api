package lingchuang

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

// TaskAdaptor implements the task-based channel for LingchuangAI (灵创 AI).
// It supports asynchronous video generation (POST /v1/video/generations) and
// polling (GET /v1/video/generations/{id}). Image generation on LingchuangAI
// is synchronous and is handled by the standard OpenAI-compatible image flow.
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
	return fmt.Sprintf("%s/v1/video/generations", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	// Idempotency-Key is required by LingchuangAI for video submissions.
	// Use the public task ID as the idempotency key so retries keep the same key.
	idempotencyKey := info.PublicTaskID
	if idempotencyKey == "" {
		idempotencyKey = c.GetString("request_id")
	}
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("newapi_%d", time.Now().UnixNano())
	}
	req.Header.Set("Idempotency-Key", idempotencyKey)
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

	// Reference images
	var refImages []string
	if len(req.ReferenceImages) > 0 {
		refImages = req.ReferenceImages
	} else if len(req.Images) > 0 {
		refImages = req.Images
	} else if len(req.ImageURLs) > 0 {
		refImages = req.ImageURLs
	} else if req.Image != "" {
		refImages = []string{req.Image}
	}
	if len(refImages) > 0 {
		payload["reference_image_urls"] = refImages
	}

	// Reference videos
	var refVideos []string
	if len(req.ReferenceVideo) > 0 {
		refVideos = req.ReferenceVideo
	} else if len(req.VideoURLs) > 0 {
		refVideos = req.VideoURLs
	}
	if len(refVideos) > 0 {
		payload["reference_video_urls"] = refVideos
	}

	// Reference audios
	if len(req.ReferenceAudio) > 0 {
		payload["reference_audio_urls"] = req.ReferenceAudio
	}

	// Duration: prefer Duration, fallback to metadata seconds
	duration := req.Duration
	if duration == 0 && req.Seconds != "" {
		if d, err := strconv.Atoi(req.Seconds); err == nil {
			duration = d
		}
	}
	if duration == 0 && req.Metadata != nil {
		if raw, ok := req.Metadata["duration"]; ok {
			duration = parseInt(raw)
		} else if raw, ok := req.Metadata["seconds"]; ok {
			duration = parseInt(raw)
		}
	}
	if duration > 0 {
		payload["duration"] = duration
	}

	// Aspect ratio: prefer AspectRatio, then Ratio, then metadata, then Size
	aspectRatio := req.AspectRatio
	if aspectRatio == "" && req.Ratio != "" {
		aspectRatio = req.Ratio
	}
	if aspectRatio == "" && req.Metadata != nil {
		if raw, ok := req.Metadata["aspect_ratio"]; ok {
			if s, ok := raw.(string); ok {
				aspectRatio = s
			}
		}
		if aspectRatio == "" {
			if raw, ok := req.Metadata["ratio"]; ok {
				if s, ok := raw.(string); ok {
					aspectRatio = s
				}
			}
		}
	}
	if aspectRatio == "" && req.Size != "" {
		if strings.Contains(req.Size, ":") {
			aspectRatio = req.Size
		} else {
			aspectRatio, _ = taskcommon.ParseSizeToRatio(req.Size)
		}
	}
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	payload["aspect_ratio"] = aspectRatio

	// Resolution: use quality/resolution field if provided.
	resolution := req.Resolution
	if resolution == "" && req.Metadata != nil {
		if raw, ok := req.Metadata["resolution"]; ok {
			if s, ok := raw.(string); ok {
				resolution = s
			}
		}
	}
	if resolution != "" {
		payload["resolution"] = resolution
	}

	// Optional fields from metadata.
	if req.Metadata != nil {
		knownFields := map[string]bool{
			"model": true, "prompt": true, "duration": true, "seconds": true,
			"aspect_ratio": true, "ratio": true, "size": true, "resolution": true,
			"reference_image_urls": true, "reference_video_urls": true, "reference_audio_urls": true,
			"images": true, "videos": true, "audios": true, "image_urls": true, "video_urls": true,
			"negative_prompt": true, "generate_audio": true, "watermark": true, "seed": true,
		}
		for k, v := range req.Metadata {
			if knownFields[k] {
				continue
			}
			if _, exists := payload[k]; !exists {
				payload[k] = v
			}
		}
	}

	// Negative prompt from metadata if not already set.
	if req.Metadata != nil {
		if raw, ok := req.Metadata["negative_prompt"]; ok {
			if s, ok := raw.(string); ok && s != "" {
				if _, exists := payload["negative_prompt"]; !exists {
					payload["negative_prompt"] = s
				}
			}
		}
		// generate_audio, watermark, seed.
		for _, key := range []string{"generate_audio", "watermark"} {
			if raw, ok := req.Metadata[key]; ok {
				if b, ok := raw.(bool); ok {
					payload[key] = b
				}
			}
		}
		if raw, ok := req.Metadata["seed"]; ok {
			payload["seed"] = parseInt(raw)
		}
	}

	body, err := common.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal lingchuang request body failed")
	}
	common.SysLog(fmt.Sprintf("[LingchuangAI] request body: %s", string(body)))
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// ---- Request / Response structures ----

// submitResponse matches LingchuangAI's POST /v1/video/generations response.
type submitResponse struct {
	ID          string     `json:"id"`
	Object      string     `json:"object,omitempty"`
	Status      string     `json:"status,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
	Model       string     `json:"model,omitempty"`
	ResultURL   string     `json:"result_url,omitempty"`
	CoverURL    string     `json:"cover_url,omitempty"`
	Error       *taskError `json:"error,omitempty"`
}

// queryResponse matches LingchuangAI's GET /v1/video/generations/{id} response.
type queryResponse struct {
	ID          string     `json:"id"`
	Object      string     `json:"object,omitempty"`
	Status      string     `json:"status,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
	Model       string     `json:"model,omitempty"`
	ResultURL   string     `json:"result_url,omitempty"`
	CoverURL    string     `json:"cover_url,omitempty"`
	Error       *taskError `json:"error,omitempty"`
}

type taskError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
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

	common.SysLog(fmt.Sprintf("[LingchuangAI] submit response body: %s", string(responseBody)))

	var sResp submitResponse
	if err := common.Unmarshal(responseBody, &sResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if sResp.Error != nil && sResp.Error.Message != "" {
		code := http.StatusBadRequest
		if sResp.Error.Code != "" {
			if c, err := strconv.Atoi(sResp.Error.Code); err == nil && c > 0 {
				code = c
			}
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("lingchuang error: %s", sResp.Error.Message), sResp.Error.Code, code)
		return
	}

	if sResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// LingchuangAI returns 202 Accepted for video submissions. Treat any 2xx as success.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("lingchuang returned status %d", resp.StatusCode), "upstream_error", resp.StatusCode)
		return
	}

	// Return OpenAI-compatible response to downstream client.
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return sResp.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	url := fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)
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

	if qResp.Error != nil {
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = qResp.Error.Message
		if taskInfo.Reason == "" {
			taskInfo.Reason = qResp.Error.Code
		}
		return taskInfo, nil
	}

	status := strings.ToLower(qResp.Status)
	switch status {
	case "succeeded", "completed", "success", "done":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if qResp.ResultURL != "" {
			taskInfo.Url = qResp.ResultURL
		}
	case "failed", "failure", "error":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = qResp.Status
	case "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = "cancelled"
	case "queued", "submitting", "processing":
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
		// Video models (examples; real model IDs come from GET /v1/models)
		"lc-fl-sd2",
		// Image models (examples; image generation is synchronous)
		"lc-image2",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "lingchuang"
}

// ---- helpers ----

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
