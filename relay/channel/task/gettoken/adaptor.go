package gettoken

import (
	"bytes"
	"encoding/json"
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
	Duration     string   `json:"duration"`
	Resolution   string   `json:"resolution,omitempty"`
	ImageUrls    []string `json:"imageUrls,omitempty"`
	ClientTaskId string   `json:"clientTaskId,omitempty"`
}

type createResponse struct {
	Code  int             `json:"code"`
	Data  []createResponseItem `json:"data,omitempty"`
	Msg   string          `json:"msg,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

type createResponseItem struct {
	Status string `json:"status"`
	TaskId string `json:"taskId"`
}

type gettokenError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type queryResponse struct {
	Code  int             `json:"code"`
	Data  *queryData      `json:"data,omitempty"`
	Msg   string          `json:"msg,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

type queryData struct {
	TaskId      string `json:"taskId"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	VideoUrl    string `json:"videoUrl,omitempty"`
	ImageUrl    string `json:"imageUrl,omitempty"`
	ErrorMsg    string `json:"errorMsg,omitempty"`
	CreatedAt   int64  `json:"createdAt,omitempty"`
	CompletedAt int64  `json:"completedAt,omitempty"`
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
		Duration:    "8",
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
	var rawMap map[string]any
	_ = common.Unmarshal(responseBody, &cResp)
	_ = common.Unmarshal(responseBody, &rawMap)

	// Extract taskId from various possible response formats
	if len(cResp.Data) > 0 && cResp.Data[0].TaskId != "" {
		upstreamTaskID = cResp.Data[0].TaskId
	}
	if upstreamTaskID == "" {
		// data as object: { "data": { "taskId": "xxx" } }
		if data, ok := rawMap["data"].(map[string]any); ok {
			if tid, ok := data["taskId"].(string); ok && tid != "" {
				upstreamTaskID = tid
			}
			if tid, ok := data["task_id"].(string); ok && tid != "" {
				upstreamTaskID = tid
			}
		}
		// data as array: { "data": [ { "taskId": "xxx" } ] }
		if dataArr, ok := rawMap["data"].([]any); ok && len(dataArr) > 0 {
			if item, ok := dataArr[0].(map[string]any); ok {
				if tid, ok := item["taskId"].(string); ok && tid != "" {
					upstreamTaskID = tid
				}
				if tid, ok := item["task_id"].(string); ok && tid != "" {
					upstreamTaskID = tid
				}
			}
		}
		// top-level taskId
		if tid, ok := rawMap["taskId"].(string); ok && tid != "" {
			upstreamTaskID = tid
		}
		if tid, ok := rawMap["task_id"].(string); ok && tid != "" {
			upstreamTaskID = tid
		}
	}

	// Check error
	if len(cResp.Error) > 0 {
		var errStr string
		if err := common.Unmarshal(cResp.Error, &errStr); err == nil && errStr != "" {
			return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken error: %s", errStr), "upstream_error", http.StatusBadRequest)
		}
		var errObj gettokenError
		if err := common.Unmarshal(cResp.Error, &errObj); err == nil {
			return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken error: %s", errObj.Message), errObj.Type, http.StatusBadRequest)
		}
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken error: %s", string(cResp.Error)), "upstream_error", http.StatusBadRequest)
	}
	if errMsg, ok := rawMap["error"].(string); ok && errMsg != "" {
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken error: %s", errMsg), "upstream_error", http.StatusBadRequest)
	}

	code := cResp.Code
	if code == 0 {
		if codeFloat, ok := rawMap["code"].(float64); ok {
			code = int(codeFloat)
		}
	}
	if code != 200 && code != 0 {
		msg := cResp.Msg
		if msg == "" {
			if msgStr, ok := rawMap["msg"].(string); ok {
				msg = msgStr
			}
		}
		return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("gettoken returned code %d: %s", code, msg), "upstream_error", code)
	}

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
	// GetToken task query endpoint
	url := fmt.Sprintf("%s/v1/tasks/%s", strings.TrimSuffix(baseUrl, "/"), taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
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

	if len(qResp.Error) > 0 {
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		// Try parse error as string first
		var errStr string
		if err := common.Unmarshal(qResp.Error, &errStr); err == nil && errStr != "" {
			taskInfo.Reason = errStr
		} else {
			var errObj gettokenError
			if err := common.Unmarshal(qResp.Error, &errObj); err == nil {
				taskInfo.Reason = errObj.Message
			} else {
				taskInfo.Reason = string(qResp.Error)
			}
		}
		return taskInfo, nil
	}

	if qResp.Data == nil {
		taskInfo.Status = model.TaskStatusInProgress
		return taskInfo, nil
	}

	d := qResp.Data
	if d.Progress > 0 {
		taskInfo.Progress = fmt.Sprintf("%d%%", d.Progress)
	}

	switch d.Status {
	case "completed", "success", "done":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		if d.VideoUrl != "" {
			taskInfo.Url = d.VideoUrl
		} else if d.ImageUrl != "" {
			taskInfo.Url = d.ImageUrl
		}
	case "failed", "error", "cancelled":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = d.ErrorMsg
	case "pending", "processing", "queued", "submitted":
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
	return []string{"veo3.1-fast"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "gettoken"
}
