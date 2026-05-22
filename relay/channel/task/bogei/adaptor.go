package bogei

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

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
	if ct, ok := c.Get("bogei_content_type"); ok {
		req.Header.Set("Content-Type", ct.(string))
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
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

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// model
	model := info.UpstreamModelName
	if v, ok := bodyMap["model"].(string); ok && v != "" {
		model = v
	}
	writer.WriteField("model", model)

	// prompt
	if prompt, ok := bodyMap["prompt"].(string); ok && prompt != "" {
		writer.WriteField("prompt", prompt)
	}

	// aspect_ratio / size
	aspectRatio := ""
	if v, ok := bodyMap["aspect_ratio"].(string); ok && v != "" {
		aspectRatio = v
	} else if v, ok := bodyMap["size"].(string); ok && v != "" {
		aspectRatio = mapSizeToAspectRatio(v)
	}
	if aspectRatio != "" {
		writer.WriteField("aspect_ratio", aspectRatio)
	}

	// images (URL or base64 strings)
	if images, ok := bodyMap["images"]; ok {
		if imgList, ok := images.([]interface{}); ok {
			for _, img := range imgList {
				if imgStr, ok := img.(string); ok && imgStr != "" {
					writer.WriteField("images", imgStr)
				}
			}
		} else if imgStr, ok := images.(string); ok && imgStr != "" {
			writer.WriteField("images", imgStr)
		}
	}
	if image, ok := bodyMap["image"].(string); ok && image != "" {
		writer.WriteField("images", image)
	}

	// enhance_prompt (default true)
	enhancePrompt := "true"
	if v, ok := bodyMap["enhance_prompt"].(bool); ok {
		enhancePrompt = strconv.FormatBool(v)
	} else if v, ok := bodyMap["enhance_prompt"].(string); ok {
		enhancePrompt = v
	}
	writer.WriteField("enhance_prompt", enhancePrompt)

	// enable_upsample (default true)
	enableUpsample := "true"
	if v, ok := bodyMap["enable_upsample"].(bool); ok {
		enableUpsample = strconv.FormatBool(v)
	} else if v, ok := bodyMap["enable_upsample"].(string); ok {
		enableUpsample = v
	}
	writer.WriteField("enable_upsample", enableUpsample)

	writer.Close()

	// Store content type with boundary for BuildRequestHeader
	c.Set("bogei_content_type", writer.FormDataContentType())

	return bytes.NewReader(buf.Bytes()), nil
}

func mapSizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024":
		return "1:1"
	case "1024x1792":
		return "9:16"
	case "1792x1024":
		return "16:9"
	case "1280x720":
		return "16:9"
	case "720x1280":
		return "9:16"
	default:
		return ""
	}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

type submitResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	StatusUpdateTime int64  `json:"status_update_time"`
	VideoURL         string `json:"video_url"`
	EnhancedPrompt   string `json:"enhanced_prompt"`
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
	var dResp submitResponse
	if err := common.Unmarshal(respBody, &dResp); err != nil {
		return nil, fmt.Errorf("unmarshal query response failed: %w", err)
	}

	taskInfo := &relaycommon.TaskInfo{
		Code: 0,
	}

	switch dResp.Status {
	case "success", "completed", "succeeded":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		taskInfo.Url = dResp.VideoURL
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
