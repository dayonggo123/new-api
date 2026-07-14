package lingdong

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

// TaskAdaptor implements the task-based channel for LingdongAPI (灵动API).
// It supports both image generation (POST /v1/images/generations) and video
// generation (POST /v1/video/generations) using public URL references.
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

func (a *TaskAdaptor) isImageGeneration(info *relaycommon.RelayInfo) bool {
	if strings.Contains(info.RequestURLPath, "/images/") {
		return true
	}
	modelName := strings.ToLower(info.UpstreamModelName)
	return modelName == "cvk-image-2" || strings.Contains(modelName, "image")
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.isImageGeneration(info) {
		return fmt.Sprintf("%s/v1/images/generations", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/video/generations", a.baseURL), nil
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

	isImage := a.isImageGeneration(info)

	payload := map[string]interface{}{
		"model":  info.UpstreamModelName,
		"prompt": req.Prompt,
	}

	// Collect reference images from various sources.
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
	// Fallback to metadata image_urls if present.
	if len(images) == 0 && req.Metadata != nil {
		if raw, ok := req.Metadata["image_urls"]; ok {
			images = parseStringSlice(raw)
		} else if raw, ok := req.Metadata["images"]; ok {
			images = parseStringSlice(raw)
		}
	}
	if len(images) > 0 {
		payload["images"] = images
	}

	if isImage {
		// Image generation fields.
		if req.Size != "" {
			payload["size"] = req.Size
		}
		// Quality / resolution for images (optional upstream field).
		if req.Resolution != "" {
			payload["resolution"] = req.Resolution
		}
		// n from metadata
		if req.Metadata != nil {
			if raw, ok := req.Metadata["n"]; ok {
				if n := parseInt(raw); n > 0 {
					payload["n"] = n
				}
			}
		}
	} else {
		// Video generation fields.
		var videos []string
		if len(req.VideoURLs) > 0 {
			videos = req.VideoURLs
		} else if len(req.ReferenceVideo) > 0 {
			videos = req.ReferenceVideo
		} else if req.Metadata != nil {
			if raw, ok := req.Metadata["video_urls"]; ok {
				videos = parseStringSlice(raw)
			} else if raw, ok := req.Metadata["videos"]; ok {
				videos = parseStringSlice(raw)
			}
		}
		if len(videos) > 0 {
			payload["videos"] = videos
		}

		var audios []string
		if len(req.ReferenceAudio) > 0 {
			audios = req.ReferenceAudio
		} else if req.Metadata != nil {
			if raw, ok := req.Metadata["audios"]; ok {
				audios = parseStringSlice(raw)
			} else if raw, ok := req.Metadata["audio_urls"]; ok {
				audios = parseStringSlice(raw)
			}
		}
		if len(audios) > 0 {
			payload["audios"] = audios
		}

		// Aspect ratio: prefer explicit ratio, then aspect_ratio, then size.
		ratio := req.Ratio
		if ratio == "" {
			ratio = req.AspectRatio
		}
		if ratio == "" && req.Metadata != nil {
			if raw, ok := req.Metadata["ratio"]; ok {
				if s, ok := raw.(string); ok {
					ratio = s
				}
			}
		}
	if ratio == "" && req.Size != "" {
		if strings.Contains(req.Size, ":") {
			ratio = req.Size
		} else {
			ratio, _ = taskcommon.ParseSizeToRatio(req.Size)
		}
	}
		if ratio == "" {
			ratio = "16:9"
		}
		payload["ratio"] = ratio

		// Duration
		duration := req.Duration
		if duration == 0 && req.Metadata != nil {
			if raw, ok := req.Metadata["duration"]; ok {
				duration = parseInt(raw)
			} else if raw, ok := req.Metadata["seconds"]; ok {
				duration = parseInt(raw)
			}
		}
		if duration == 0 {
			duration = 6
		}
		payload["duration"] = duration

		// Resolution
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
	}

	// Merge any remaining metadata fields that are not already handled.
	if req.Metadata != nil {
		knownFields := map[string]bool{
			"model": true, "prompt": true, "images": true, "videos": true,
			"audios": true, "image_urls": true, "video_urls": true, "audio_urls": true,
			"ratio": true, "aspect_ratio": true, "duration": true, "seconds": true,
			"resolution": true, "size": true, "n": true, "quality": true,
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

	body, err := common.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal lingdong request body failed")
	}
	common.SysLog(fmt.Sprintf("[LingdongAPI] request body: %s", string(body)))
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// ---- Request / Response structures ----

type createResponse struct {
	Code  int                    `json:"code"`
	Data  []createResponseItem   `json:"data,omitempty"`
	Error *lingdongError         `json:"error,omitempty"`
}

type createResponseItem struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type lingdongError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type queryResponse struct {
	Code  int       `json:"code"`
	Data  queryData `json:"data,omitempty"`
	Error *lingdongError `json:"error,omitempty"`
}

type queryData struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	URL      string `json:"url,omitempty"`
	Result   *queryResult `json:"result,omitempty"`
	Error    *lingdongError `json:"error,omitempty"`
}

type queryResult struct {
	Images []string `json:"images,omitempty"`
	Videos []string `json:"videos,omitempty"`
	URL    string   `json:"url,omitempty"`
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

	var cResp createResponse
	if err := common.Unmarshal(responseBody, &cResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	common.SysLog(fmt.Sprintf("[LingdongAPI] create response body: %s", string(responseBody)))

	if cResp.Error != nil {
		codeInt := 400
		if code, err := strconv.Atoi(cResp.Error.Code); err == nil {
			codeInt = code
		}
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("lingdong error: %s", cResp.Error.Message), cResp.Error.Type, codeInt)
		return
	}

	if cResp.Code != 200 {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("lingdong returned code %d", cResp.Code), "upstream_error", cResp.Code)
		return
	}

	if len(cResp.Data) == 0 || cResp.Data[0].TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// Return OpenAI-compatible response to client.
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return cResp.Data[0].TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	url := fmt.Sprintf("%s/v1/video/generations/%s", baseUrl, taskID)
	if a.isImageGenerationFromBody(body) {
		url = fmt.Sprintf("%s/v1/images/generations/%s", baseUrl, taskID)
	}
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

func (a *TaskAdaptor) isImageGenerationFromBody(body map[string]any) bool {
	if body == nil {
		return false
	}
	if modelName, ok := body["model"].(string); ok {
		lower := strings.ToLower(modelName)
		return lower == "cvk-image-2" || strings.Contains(lower, "image")
	}
	return false
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
		return taskInfo, nil
	}

	d := qResp.Data

	if d.Progress > 0 {
		taskInfo.Progress = fmt.Sprintf("%d%%", d.Progress)
	}

	status := strings.ToLower(d.Status)
	switch status {
	case "completed", "success", "done":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if d.URL != "" {
			taskInfo.Url = d.URL
		} else if d.Result != nil {
			if d.Result.URL != "" {
				taskInfo.Url = d.Result.URL
			} else if len(d.Result.Videos) > 0 {
				taskInfo.Url = d.Result.Videos[0]
			} else if len(d.Result.Images) > 0 {
				taskInfo.Url = d.Result.Images[0]
			}
		}
	case "failed", "failure", "error":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		if d.Error != nil && d.Error.Message != "" {
			taskInfo.Reason = d.Error.Message
		} else {
			taskInfo.Reason = d.Status
		}
	case "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = "cancelled"
	case "submitted", "pending", "processing", "queued":
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
		// Image
		"cvk-image-2",
		// Video
		"cvk-video-2",
		"cvk-2-1",
		"cvk-2-2",
		"cvk-2-4",
		"cvk-2-7",
		"cvk-2-11",
		"cvk",
		"cvk-2-17",
		"cvk-2-fast-720",
		"cvk-3",
		"cvk-4",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "lingdong"
}

// ---- helpers ----

func parseStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		var out []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if val != "" {
			return []string{val}
		}
	}
	return nil
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
