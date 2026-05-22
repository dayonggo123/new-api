package bogei

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
)

// TaskAdaptor implements the task platform interface for BogeiAI (波哥AI站).
type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + "/v1/videos", nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, err
	}

	var bodyMap map[string]interface{}
	if err := common.Unmarshal(cachedBody, &bodyMap); err != nil {
		return bytes.NewReader(cachedBody), nil
	}

	// Fallback: build from task_request context (for channel test etc.)
	if len(bodyMap) == 0 {
		if taskReq, err := relaycommon.GetTaskRequest(c); err == nil && taskReq.Prompt != "" {
			bodyMap = map[string]interface{}{
				"prompt": taskReq.Prompt,
				"model":  info.UpstreamModelName,
			}
			if taskReq.Size != "" {
				bodyMap["size"] = taskReq.Size
			}
		}
	}

	bogeiBody := make(map[string]interface{})

	// model: prefer info.UpstreamModelName, fallback to bodyMap, then info.OriginModelName
	model := info.UpstreamModelName
	if model == "" {
		if m, ok := bodyMap["model"].(string); ok && m != "" {
			model = m
		}
	}
	if model == "" {
		model = info.OriginModelName
	}
	if model != "" {
		bogeiBody["model"] = mapModelName(model)
	}

	// prompt
	prompt := ""
	if p, ok := bodyMap["prompt"].(string); ok && p != "" {
		prompt = p
	}
	if prompt == "" {
		if taskReq, err := relaycommon.GetTaskRequest(c); err == nil {
			prompt = taskReq.Prompt
		}
	}
	if prompt != "" {
		bogeiBody["prompt"] = prompt
	}

	// aspect_ratio / size
	aspectRatio := ""
	if v, ok := bodyMap["aspect_ratio"].(string); ok && v != "" {
		aspectRatio = strings.TrimSpace(v)
	} else if v, ok := bodyMap["size"].(string); ok && v != "" {
		aspectRatio = mapSizeToAspectRatio(strings.TrimSpace(v))
	}
	if aspectRatio != "" {
		bogeiBody["aspect_ratio"] = aspectRatio
	}

	// images (URL strings)
	var images []string
	if imgs, ok := bodyMap["images"]; ok {
		if imgList, ok := imgs.([]interface{}); ok {
			for _, img := range imgList {
				if imgStr, ok := img.(string); ok && imgStr != "" {
					images = append(images, imgStr)
				}
			}
		} else if imgStr, ok := imgs.(string); ok && imgStr != "" {
			images = append(images, imgStr)
		}
	}
	if img, ok := bodyMap["image"].(string); ok && img != "" {
		images = append(images, img)
	}
	if len(images) > 0 {
		bogeiBody["images"] = images
	}

	// enhance_prompt (default true)
	enhancePrompt := true
	if v, ok := bodyMap["enhance_prompt"].(bool); ok {
		enhancePrompt = v
	} else if v, ok := bodyMap["enhance_prompt"].(string); ok {
		enhancePrompt = strings.ToLower(v) == "true"
	}
	bogeiBody["enhance_prompt"] = enhancePrompt

	// enable_upsample (default true) — upstream accepts boolean
	enableUpsample := true
	if v, ok := bodyMap["enable_upsample"].(bool); ok {
		enableUpsample = v
	} else if v, ok := bodyMap["enable_upsample"].(string); ok {
		enableUpsample = strings.ToLower(v) == "true"
	}
	bogeiBody["enable_upsample"] = enableUpsample

	jsonData, err := common.Marshal(bogeiBody)
	if err != nil {
		return nil, err
	}
	common.SysLog(fmt.Sprintf("[BogeiAI] upstream request body: %s", string(jsonData)))
	return bytes.NewReader(jsonData), nil
}

func mapModelName(model string) string {
	// Strip provider prefix if any
	if idx := strings.Index(model, "/"); idx >= 0 {
		model = model[idx+1:]
	}
	switch model {
	case "veo-3.1-fast", "veo_3_1-fast":
		return "veo_3_1-components"
	case "veo-3.1", "veo_3_1":
		return "veo_3_1-components"
	default:
		return model
	}
}

func mapSizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
	case "1024x1792", "720x1280", "1080x1920":
		return "9:16"
	case "1792x1024", "1280x720", "1920x1080":
		return "16:9"
	default:
		return "16:9"
	}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// submitResponse matches BogeiAI's submit API response.
type submitResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	StatusUpdateTime int64  `json:"status_update_time"`
}

// queryResponse matches BogeiAI's query API response.
// It may contain either a flat structure or a nested "detail" object.
type queryResponse struct {
	ID               string      `json:"id"`
	Status           string      `json:"status"`
	VideoURL         string      `json:"video_url"`
	EnhancedPrompt   string      `json:"enhanced_prompt"`
	StatusUpdateTime int64       `json:"status_update_time"`
	Detail           *taskDetail `json:"detail"`
}

type taskDetail struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	VideoURL string `json:"video_url"`
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var dResp submitResponse
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("body: %s", string(responseBody)), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task id is empty, response: %s", string(responseBody)), "invalid_response", http.StatusInternalServerError)
		return
	}

	// Write OpenAI Video API format response to downstream client
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	if ov.Model == "" {
		ov.Model = info.UpstreamModelName
	}
	c.JSON(http.StatusOK, ov)

	return dResp.ID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := baseUrl + "/v1/videos/" + taskID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var qResp queryResponse
	if err := common.Unmarshal(respBody, &qResp); err != nil {
		return nil, fmt.Errorf("unmarshal query response failed: %w", err)
	}

	taskInfo := &relaycommon.TaskInfo{
		Code: 0,
	}

	// Use nested detail.status if available, otherwise fall back to top-level status
	status := qResp.Status
	if qResp.Detail != nil && qResp.Detail.Status != "" {
		status = qResp.Detail.Status
	}

	// Use nested detail.video_url if available
	videoURL := qResp.VideoURL
	if qResp.Detail != nil && qResp.Detail.VideoURL != "" {
		videoURL = qResp.Detail.VideoURL
	}

	switch status {
	case "success", "completed", "succeeded":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		taskInfo.Url = videoURL
	case "failed", "failure":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
	default:
		taskInfo.Status = model.TaskStatusInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"default"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "bogei"
}
