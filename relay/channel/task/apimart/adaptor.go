package apimart

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
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

// ---- Create (Image) ----

type imageCreateRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	N                int      `json:"n,omitempty"`
	Size             string   `json:"size,omitempty"`
	AspectRatio      string   `json:"aspect_ratio,omitempty"`
	Resolution       string   `json:"resolution,omitempty"`
	ImageURLs        []string `json:"image_urls,omitempty"`
	OfficialFallback bool     `json:"official_fallback,omitempty"`
}

// ---- Create (Video) ----

type videoCreateRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	Duration         int      `json:"duration,omitempty"`
	AspectRatio      string   `json:"aspect_ratio,omitempty"`
	GenerationType   string   `json:"generation_type,omitempty"`
	ImageURLs        []string `json:"image_urls,omitempty"`
	Resolution       string   `json:"resolution,omitempty"`
	EnableGIF        bool     `json:"enable_gif,omitempty"`
	OfficialFallback bool     `json:"official_fallback,omitempty"`
}

// ---- Create Response ----

type createResponse struct {
	Code  int                   `json:"code"`
	Data  []createResponseItem  `json:"data,omitempty"`
	Error *apimartError         `json:"error,omitempty"`
}

type createResponseItem struct {
	Status  string `json:"status"`
	TaskID  string `json:"task_id"`
}

type apimartError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ---- Query Response ----

type queryResponse struct {
	Code  int           `json:"code"`
	Data  *queryData    `json:"data,omitempty"`
	Error *apimartError `json:"error,omitempty"`
}

type queryData struct {
	ID               string           `json:"id"`
	Status           string           `json:"status"`
	Progress         int              `json:"progress"`
	Result           *queryResult     `json:"result,omitempty"`
	Created          int64            `json:"created"`
	Completed        int64            `json:"completed,omitempty"`
	EstimatedTime    int              `json:"estimated_time,omitempty"`
	ActualTime       int              `json:"actual_time,omitempty"`
	Error            *apimartError    `json:"error,omitempty"`
}

type queryResult struct {
	Images       []resultItem `json:"images,omitempty"`
	Videos       []resultItem `json:"videos,omitempty"`
	ThumbnailURL string       `json:"thumbnail_url,omitempty"`
}

type resultItem struct {
	URL       []string `json:"url"`
	ExpiresAt int64    `json:"expires_at"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "get_task_request_failed", http.StatusBadRequest)
	}
	info.Action = constant.TaskActionGenerate
	if metaAction, ok := req.Metadata["action"]; ok {
		if actionStr, ok := metaAction.(string); ok && actionStr != "" {
			info.Action = actionStr
		}
	}
	return nil
}

func (a *TaskAdaptor) isImageGeneration(info *relaycommon.RelayInfo) bool {
	if strings.Contains(info.RequestURLPath, "/images/") {
		return true
	}
	if strings.HasPrefix(info.UpstreamModelName, "gpt-image") {
		return true
	}
	return false
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.isImageGeneration(info) {
		return fmt.Sprintf("%s/v1/images/generations", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos/generations", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	common.SysError(fmt.Sprintf("[APIMart] BuildRequestBody: prompt=%q size=%q aspect_ratio=%q images=%d imageURLs=%d referenceImages=%d metadata=%v", req.Prompt, req.Size, req.AspectRatio, len(req.Images), len(req.ImageURLs), len(req.ReferenceImages), req.Metadata))

	body, err := a.convertToRequestPayload(req, info)
	if err != nil {
		return nil, err
	}
	common.SysError(fmt.Sprintf("[APIMart] request body: %s", string(body)))
	c.Set("apimart_request_body", string(body))
	return bytes.NewReader(body), nil
}

// mapSizeToAspectRatio 将常见 size 值映射为 APIMart 支持的宽高比或像素串。
// APIMart 支持：1:1 / 16:9 / 9:16 / 4:3 / 3:4 / 3:2 / 2:3 / 5:4 / 4:5 / 2:1 / 1:2 / 21:9 / 9:21 / 3:1 / 1:3（或直接传像素串）
func mapSizeToAspectRatio(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	if size == "" {
		return "1:1"
	}
	// 已经是宽高比格式
	if strings.Contains(size, ":") {
		return size
	}
	// 常见 OpenAI 像素映射
	switch size {
	case "1024x1024", "1024", "1k":
		return "1:1"
	case "1024x1792", "1792x1024":
		// 需要根据具体值判断，后面处理
	}
	// 如果包含 x，当作像素串直接传（APIMart 支持像素串）
	if strings.Contains(size, "x") {
		return size
	}
	// fallback
	return "1:1"
}

// mapAspectRatioToPixelSize 把常见比例映射为 OpenAI gpt-image-2 支持的像素尺寸。
// gpt-image-2 支持：1024x1024 / 1024x1536 / 1536x1024
func mapAspectRatioToPixelSize(ratio string) string {
	ratio = strings.ToLower(strings.TrimSpace(ratio))
	switch ratio {
	case "1:1", "1/1":
		return "1024x1024"
	case "9:16", "2:3", "3:4", "4:5", "5:8", "10:16", "3:5":
		return "1024x1536"
	case "16:9", "3:2", "4:3", "16:10", "21:9", "2:1", "5:4", "5:3":
		return "1536x1024"
	default:
		// 尝试按冒号解析宽高比，粗略判断横竖屏
		if strings.Contains(ratio, ":") {
			parts := strings.Split(ratio, ":")
			if len(parts) == 2 {
				w, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
				h, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
				if err1 == nil && err2 == nil && h > 0 {
					if w/h < 1.0 {
						return "1024x1536"
					}
					return "1536x1024"
				}
			}
		}
		return ratio // fallback：原样返回（可能是像素串）
	}
}

func (a *TaskAdaptor) convertToRequestPayload(req relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) ([]byte, error) {
	isImage := a.isImageGeneration(info)

	// Collect image URLs from req.Images / req.ImageURLs / req.ReferenceImages / req.Image
	imageURLs := req.Images
	if len(imageURLs) == 0 {
		imageURLs = req.ImageURLs
	}
	if len(imageURLs) == 0 {
		imageURLs = req.ReferenceImages
	}
	if len(imageURLs) == 0 && req.Image != "" {
		imageURLs = []string{req.Image}
	}

	// Collect aspect ratio from req.AspectRatio / metadata["aspect_ratio"] / req.Size
	aspectRatio := ""
	if req.AspectRatio != "" {
		aspectRatio = req.AspectRatio
	} else if req.Metadata != nil {
		if v, ok := req.Metadata["aspect_ratio"].(string); ok && v != "" {
			aspectRatio = v
		}
	}
	if aspectRatio == "" && strings.Contains(req.Size, ":") {
		aspectRatio = req.Size
	}

	if isImage {
		payload := imageCreateRequest{
			Model:  info.UpstreamModelName,
			Prompt: req.Prompt,
			N:      1,
		}

		// 设置尺寸：优先用像素串（如果 req.Size 是像素格式），
		// 否则把 aspectRatio 映射为像素尺寸；兜底 1:1
		if req.Size != "" && strings.Contains(req.Size, "x") {
			payload.Size = req.Size
		} else if aspectRatio != "" {
			payload.AspectRatio = aspectRatio
			payload.Size = mapAspectRatioToPixelSize(aspectRatio)
		} else {
			payload.Size = "1024x1024"
		}

		if len(imageURLs) > 0 {
			payload.ImageURLs = imageURLs
		}

		// Metadata overrides
		if req.Metadata != nil {
			if v, ok := req.Metadata["resolution"].(string); ok && v != "" {
				payload.Resolution = v
			} else {
				payload.Resolution = "1k"
			}
			if v, ok := req.Metadata["n"].(float64); ok {
				payload.N = int(v)
			}
			if v, ok := req.Metadata["official_fallback"].(bool); ok {
				payload.OfficialFallback = v
			}
		} else {
			payload.Resolution = "1k"
		}

		body, _ := common.Marshal(payload)
		common.SysLog(fmt.Sprintf("[APIMart] image request payload: %s", string(body)))
		return body, nil
	}

	// Video generation
	payload := videoCreateRequest{
		Model:  info.UpstreamModelName,
		Prompt: req.Prompt,
	}
	common.SysLog(fmt.Sprintf("[APIMart] video payload before marshal: model=%s prompt=%q duration=%d aspect_ratio=%s resolution=%s", payload.Model, payload.Prompt, payload.Duration, payload.AspectRatio, payload.Resolution))

	if req.Duration > 0 {
		payload.Duration = req.Duration
	} else {
		payload.Duration = 8
	}

	if len(imageURLs) > 0 {
		payload.ImageURLs = imageURLs
	}

	// aspect_ratio from collected value, req.Size, or default
	if aspectRatio != "" {
		payload.AspectRatio = aspectRatio
	} else if req.Size != "" {
		payload.AspectRatio = mapSizeToAspectRatio(req.Size)
	} else {
		payload.AspectRatio = "16:9"
	}

	if req.Metadata != nil {
		if v, ok := req.Metadata["generation_type"].(string); ok && v != "" {
			payload.GenerationType = v
		} else if len(imageURLs) > 0 {
			// Auto-detect: 2 images = frame (first/last), 3 images = reference
			if len(imageURLs) == 2 {
				payload.GenerationType = "frame"
			} else if len(imageURLs) >= 3 {
				payload.GenerationType = "reference"
			}
		}
		if v, ok := req.Metadata["resolution"].(string); ok && v != "" {
			payload.Resolution = strings.ToLower(v)
		} else {
			payload.Resolution = "720p"
		}
		if v, ok := req.Metadata["enable_gif"].(bool); ok {
			payload.EnableGIF = v
		}
		if v, ok := req.Metadata["official_fallback"].(bool); ok {
			payload.OfficialFallback = v
		}
	} else {
		if req.Size != "" {
			payload.AspectRatio = mapSizeToAspectRatio(req.Size)
		}
		payload.Resolution = "720p"
	}

	body, _ := common.Marshal(payload)
	common.SysLog(fmt.Sprintf("[APIMart] video request payload: %s", string(body)))
	return body, nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
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

	common.SysLog(fmt.Sprintf("[APIMart] create response body: %s", string(responseBody)))

	// Handle upstream error
	if cResp.Error != nil {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("apimart error: %s", cResp.Error.Message), cResp.Error.Type, cResp.Error.Code)
		return
	}

	if cResp.Code != 200 {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("apimart returned code %d", cResp.Code), "upstream_error", cResp.Code)
		return
	}

	if len(cResp.Data) == 0 || cResp.Data[0].TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// Return OpenAI-compatible response to client
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
	if !ok {
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

	// Handle upstream error wrapper
	if qResp.Error != nil {
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = qResp.Error.Message
		return taskInfo, nil
	}

	if qResp.Data == nil {
		taskInfo.Status = model.TaskStatusInProgress
		return taskInfo, nil
	}

	d := qResp.Data

	// Progress
	if d.Progress > 0 {
		taskInfo.Progress = fmt.Sprintf("%d%%", d.Progress)
	}

	// Status mapping
	switch d.Status {
	case "completed":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if d.Result != nil {
			// Try image URL first
			if len(d.Result.Images) > 0 && len(d.Result.Images[0].URL) > 0 {
				taskInfo.Url = d.Result.Images[0].URL[0]
			}
			// Then video URL
			if taskInfo.Url == "" && len(d.Result.Videos) > 0 && len(d.Result.Videos[0].URL) > 0 {
				taskInfo.Url = d.Result.Videos[0].URL[0]
			}
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		if d.Error != nil {
			taskInfo.Reason = d.Error.Message
		} else {
			taskInfo.Reason = d.Status
		}
	case "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = "cancelled"
	case "pending", "processing":
		taskInfo.Status = model.TaskStatusInProgress
	default:
		taskInfo.Status = model.TaskStatusInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var cResp createResponse
	if err := common.Unmarshal(task.Data, &cResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task data failed")
	}

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
		"gpt-image-2",
		// Video
		"veo3.1-fast",
		"veo3.1-quality",
		"veo3.1-lite",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "apimart"
}
