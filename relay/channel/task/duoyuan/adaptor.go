package duoyuan

import (
	"bytes"
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

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type createRequest struct {
	Images          []string `json:"images,omitempty"`
	Model           string   `json:"model"`
	Orientation     string   `json:"orientation"`
	Prompt          string   `json:"prompt"`
	Size            string   `json:"size"`
	Duration        int      `json:"duration"`
	AspectRatio     string   `json:"aspect_ratio"`
	EnableUpsample  *bool    `json:"enable_upsample,omitempty"`
	EnhancePrompt   *bool    `json:"enhance_prompt,omitempty"`
	VeoFlClose      bool     `json:"veo_fl_close,omitempty"`
}

type createResponse struct {
	ID               string `json:"id"`
	Object           string `json:"object"`
	Model            string `json:"model"`
	Status           string `json:"status"`
	Progress         int    `json:"progress"`
	CreatedAt        int64  `json:"created_at"`
	Size             string `json:"size"`
	StatusUpdateTime int64  `json:"status_update_time"`
	Detail           struct {
		Input struct {
			AspectRatio string   `json:"aspect_ratio"`
			Images      []string `json:"images"`
			Model       string   `json:"model"`
			Prompt      string   `json:"prompt"`
		} `json:"input"`
	} `json:"detail"`
}

type queryResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	VideoURL         string `json:"video_url"`
	EnhancedPrompt   string `json:"enhanced_prompt"`
	StatusUpdateTime int64  `json:"status_update_time"`
	Detail           struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Status string `json:"status"`
		Input  struct {
			Size        string   `json:"size"`
			Model       string   `json:"model"`
			Images      []string `json:"images"`
			Prompt      string   `json:"prompt"`
			Orientation string   `json:"orientation"`
		} `json:"input"`
		GifURL       string `json:"gif_url,omitempty"`
		ThumbnailURL string `json:"thumbnail_url,omitempty"`
	} `json:"detail"`
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

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/video/create", a.baseURL), nil
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

	body := a.convertToRequestPayload(req, info)
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) convertToRequestPayload(req relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) *createRequest {
	payload := &createRequest{
		Model:   mapClientModelToUpstream(info.UpstreamModelName),
		Prompt:  req.Prompt,
		VeoFlClose: false,
	}

	// Images
	images := req.Images
	if len(images) == 0 && req.Image != "" {
		images = []string{req.Image}
	}
	payload.Images = images

	// Orientation & AspectRatio & Size
	aspectRatio := ""
	size := req.Size
	if size == "" {
		size = "720x1280"
	}

	// Try to get aspect_ratio from metadata first
	if req.Metadata != nil {
		if ar, ok := req.Metadata["aspect_ratio"].(string); ok && ar != "" {
			aspectRatio = ar
		}
		if s, ok := req.Metadata["size"].(string); ok && s != "" {
			size = s
		}
	}

	if aspectRatio == "" {
		aspectRatio = inferAspectRatioFromSize(size)
	}
	payload.AspectRatio = aspectRatio
	payload.Orientation = orientationFromAspectRatio(aspectRatio)
	payload.Size = size

	// Duration
	if req.Duration > 0 {
		payload.Duration = req.Duration
	} else {
		payload.Duration = 8
	}

	// Metadata overrides
	if req.Metadata != nil {
		if v, ok := req.Metadata["enable_upsample"].(bool); ok {
			payload.EnableUpsample = &v
		}
		if v, ok := req.Metadata["enhance_prompt"].(bool); ok {
			payload.EnhancePrompt = &v
		} else {
			// Default true for Veo models
			v := true
			payload.EnhancePrompt = &v
		}
		if v, ok := req.Metadata["veo_fl_close"].(bool); ok {
			payload.VeoFlClose = v
		}
	} else {
		// Default enhance_prompt = true
		v := true
		payload.EnhancePrompt = &v
	}

	return payload
}

func mapClientModelToUpstream(clientModel string) string {
	// Map common client naming conventions to upstream snake_case naming.
	// Examples:
	//   veo3.1 → veo_3_1
	//   veo3.1-fast-components → veo_3_1-fast-components
	//   veo-3.1-fast → veo_3_1-fast
	m := clientModel
	m = strings.ReplaceAll(m, "veo3.1", "veo_3_1")
	m = strings.ReplaceAll(m, "veo-3.1", "veo_3_1")
	m = strings.ReplaceAll(m, "veo3_1", "veo_3_1")
	m = strings.ReplaceAll(m, "veo-3_1", "veo_3_1")
	return m
}

func inferAspectRatioFromSize(size string) string {
	switch size {
	case "1280x720", "1920x1080":
		return "16:9"
	case "720x1280", "1080x1920":
		return "9:16"
	default:
		return "16:9"
	}
}

func orientationFromAspectRatio(ar string) string {
	switch ar {
	case "16:9", "4:3", "21:9", "1.91:1":
		return "landscape"
	case "9:16", "3:4", "1:1":
		return "portrait"
	default:
		return "landscape"
	}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
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

	if cResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// Return OpenAI-compatible video response to client
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return cResp.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s/v1/video/query?id=%s", baseUrl, taskID)
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

	// Status mapping
	switch qResp.Status {
	case "completed":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if qResp.VideoURL != "" {
			taskInfo.Url = qResp.VideoURL
		} else if qResp.Detail.URL != "" {
			taskInfo.Url = qResp.Detail.URL
		}
	case "failed", "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = qResp.Status
	case "queued", "pending", "processing":
		taskInfo.Status = model.TaskStatusInProgress
		// Try to extract progress from detail if available
		if qResp.Detail.Status != "" {
			// Some platforms include a percentage in the detail status
		}
	default:
		// Unknown status → treat as in-progress
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

	// URL from task result (set by relay_task.go after ParseTaskResult)
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
		"veo_3_1",
		"veo_3_1-4K",
		"veo_3_1-fast",
		"veo_3_1-fast-4K",
		"veo_3_1-components",
		"veo_3_1-components-4K",
		"veo_3_1-fast-components",
		"veo_3_1-fast-components-4K",
		// Client-friendly aliases (mapped automatically)
		"veo3.1",
		"veo3.1-4K",
		"veo3.1-fast",
		"veo3.1-fast-4K",
		"veo3.1-components",
		"veo3.1-components-4K",
		"veo3.1-fast-components",
		"veo3.1-fast-components-4K",
		"veo-3.1",
		"veo-3.1-fast",
		"veo-3.1-components",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "duoyuan"
}
