package gettoken

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
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Request / Response structures

type videoCreateRequest struct {
	Prompt       string   `json:"prompt"`
	AspectRatio  string   `json:"aspectRatio"`
	Duration     string   `json:"duration,omitempty"`
	Resolution   string   `json:"resolution"`
	ImageUrls    []string `json:"imageUrls,omitempty"`
	ClientTaskId string   `json:"clientTaskId,omitempty"`
	WebhookUrl   string   `json:"webhookUrl,omitempty"`
}

// GetToken create response: {taskId, status, errorCode, errorMessage, results, clientId, promptTips}
type createResponse struct {
	TaskId       string `json:"taskId"`
	Status       string `json:"status"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	Results      any    `json:"results"`
	ClientId     string `json:"clientId"`
	PromptTips   string `json:"promptTips"`
}

// GetToken query request: POST /v1/query {taskId}
type queryRequest struct {
	TaskId string `json:"taskId"`
}

// GetToken query response
type queryResponse struct {
	TaskId       string       `json:"taskId"`
	Status       string       `json:"status"`
	ErrorCode    string       `json:"errorCode"`
	ErrorMessage string       `json:"errorMessage"`
	FailedReason any          `json:"failedReason"`
	Results      []resultItem `json:"results"`
	ClientId     string       `json:"clientId"`
	PromptTips   string       `json:"promptTips"`
}

type resultItem struct {
	URL        string `json:"url"`
	OutputType string `json:"outputType"`
	Text       string `json:"text"`
}

// Adaptor implementation

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	baseURL     string
	hasImages   bool // set during ValidateRequestAndSetAction
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if err := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); err != nil {
		return err
	}
	req, _ := relaycommon.GetTaskRequest(c)
	a.hasImages = len(req.Images) > 0 || req.Image != ""
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = "veo3.1-fast"
	}
	if a.hasImages {
		return fmt.Sprintf("%s/v1/%s/image-to-video", strings.TrimSuffix(a.baseURL, "/"), modelName), nil
	}
	return fmt.Sprintf("%s/v1/%s/text-to-video", strings.TrimSuffix(a.baseURL, "/"), modelName), nil
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
	body, err := a.convertToRequestPayload(req, info)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func mapSizeToAspectRatio(size string) string {
	switch size {
	case "1792x1024", "1024x576":
		return "16:9"
	case "1024x1792", "576x1024":
		return "9:16"
	case "16:9", "9:16":
		return size
	default:
		return "16:9"
	}
}

func (a *TaskAdaptor) convertToRequestPayload(req relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) ([]byte, error) {
	imageURLs := req.Images
	if len(imageURLs) == 0 && req.Image != "" {
		imageURLs = []string{req.Image}
	}

	payload := videoCreateRequest{
		Prompt:      req.Prompt,
		AspectRatio: "16:9",
		Resolution:  "720p",
	}

	if req.Duration > 0 {
		payload.Duration = strconv.Itoa(req.Duration)
	}
	if req.Size != "" {
		payload.AspectRatio = mapSizeToAspectRatio(req.Size)
	}

	if req.Metadata != nil {
		if v, ok := req.Metadata["aspect_ratio"].(string); ok && v != "" {
			payload.AspectRatio = mapSizeToAspectRatio(v)
		}
		if v, ok := req.Metadata["resolution"].(string); ok && v != "" {
			payload.Resolution = v
		}
		if v, ok := req.Metadata["duration"].(string); ok && v != "" {
			payload.Duration = v
		}
		if v, ok := req.Metadata["webhook_url"].(string); ok && v != "" {
			payload.WebhookUrl = v
		}
	}

	if len(imageURLs) > 0 {
		payload.ImageUrls = imageURLs
	}
	if info.PublicTaskID != "" {
		payload.ClientTaskId = info.PublicTaskID
	}

	body, _ := common.Marshal(payload)
	common.SysLog(fmt.Sprintf("[GetToken] request payload: %s", string(body)))
	return body, nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	url, err := a.BuildRequestURL(info)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, requestBody)
	if err != nil {
		return nil, err
	}
	if err := a.BuildRequestHeader(c, req, info); err != nil {
		return nil, err
	}
	client := service.GetHttpClient()
	return client.Do(req)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	var upstreamTaskID string
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp == nil {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("upstream response is nil"), "upstream_nil", http.StatusInternalServerError)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	common.SysLog(fmt.Sprintf("[GetToken] create response body: %s", string(responseBody)))

	var cResp createResponse
	if err := common.Unmarshal(responseBody, &cResp); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
	}

	if cResp.ErrorCode != "" || cResp.ErrorMessage != "" {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken error [%s]: %s", cResp.ErrorCode, cResp.ErrorMessage), "upstream_error", http.StatusBadRequest)
	}

	upstreamTaskID = cResp.TaskId
	if upstreamTaskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	// GetToken query endpoint: POST /v1/query
	url := fmt.Sprintf("%s/v1/query", strings.TrimSuffix(baseUrl, "/"))
	queryBody, _ := common.Marshal(queryRequest{TaskId: taskID})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(queryBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client := service.GetHttpClient()
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

	if qResp.ErrorCode != "" || qResp.ErrorMessage != "" {
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = fmt.Sprintf("[%s] %s", qResp.ErrorCode, qResp.ErrorMessage)
		return taskInfo, nil
	}

	switch qResp.Status {
	case "SUCCESS":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if len(qResp.Results) > 0 && qResp.Results[0].URL != "" {
			taskInfo.Url = qResp.Results[0].URL
		}
	case "FAILED", "TIMEOUT":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		if qResp.ErrorMessage != "" {
			taskInfo.Reason = qResp.ErrorMessage
		}
	case "QUEUED", "RUNNING":
		taskInfo.Status = model.TaskStatusInProgress
	default:
		taskInfo.Status = model.TaskStatusInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var qResp queryResponse
	if err := common.Unmarshal(task.Data, &qResp); err != nil {
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
	return []string{"veo3.1-fast"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "gettoken"
}
