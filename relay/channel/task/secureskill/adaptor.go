package secureskill

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
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// TaskAdaptor implements the task-based channel for SecureSkill video generation.
// It supports omni (text-to-video) and omni_video_edit (video-to-video) models.
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
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionTextGenerate)
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + "/v1/chat/completions", nil
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

	common.SysLog(fmt.Sprintf("[SecureSkill] TaskSubmitReq: prompt=%q, videoURLs=%d, refVideos=%d, images=%d",
		req.Prompt, len(req.VideoURLs), len(req.ReferenceVideo), len(req.Images)))

	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = req.Model
	}

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]interface{}{
			{"role": "user", "content": req.Prompt},
		},
	}

	// Handle video reference for omni_video_edit model
	var videoURL string
	if len(req.VideoURLs) > 0 {
		videoURL = req.VideoURLs[0]
	} else if len(req.ReferenceVideo) > 0 {
		videoURL = req.ReferenceVideo[0]
	}
	if videoURL != "" {
		payload["video_url"] = videoURL
		common.SysLog(fmt.Sprintf("[SecureSkill] video_url=%s", videoURL))
	}

	// Handle generation config from metadata
	if req.Metadata != nil {
		if aspectRatio, ok := req.Metadata["aspect_ratio"].(string); ok && aspectRatio != "" {
			payload["aspect_ratio"] = aspectRatio
		}
		if duration, ok := req.Metadata["duration"].(float64); ok && duration > 0 {
			payload["duration"] = int(duration)
		}
	}

	// Handle aspect_ratio from standard field
	if req.AspectRatio != "" {
		payload["aspect_ratio"] = req.AspectRatio
	}

	// Handle duration from standard field
	if req.Duration > 0 {
		payload["duration"] = req.Duration
	}

	jsonData, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	common.SysLog(fmt.Sprintf("[SecureSkill] upstream request body: %s", string(jsonData)))
	c.Set("secureskill_request_body", string(jsonData))
	return bytes.NewReader(jsonData), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// submitResponse represents the initial submit response from SecureSkill API
type submitResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Created   int64  `json:"created"`
	Prompt    string `json:"prompt"`
	StatusURL string `json:"status_url"`
	QueueDepth int   `json:"queue_depth,omitempty"`
}

// jobResponse represents the job status polling response from SecureSkill API
type jobResponse struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	Model       string         `json:"model"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	Progress    int            `json:"progress"`
	Prompt      string         `json:"prompt"`
	StatusURL   string         `json:"status_url"`
	Result      *jobResult     `json:"result,omitempty"`
	URLs        []string       `json:"urls,omitempty"`
	URL         string         `json:"url,omitempty"`
	Choices     []jobChoice    `json:"choices,omitempty"`
	Error       *jobError      `json:"error,omitempty"`
}

type jobResult struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type jobChoice struct {
	Index int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type jobError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	common.SysLog(fmt.Sprintf("[SecureSkill] DoResponse body: %s", string(responseBody)))

	var dResp submitResponse
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		// Check if it's an error response
		var errResp map[string]interface{}
		if err2 := common.Unmarshal(responseBody, &errResp); err2 == nil {
			if errMsg, ok := errResp["message"].(string); ok {
				taskErr = service.TaskErrorWrapper(fmt.Errorf("%s", errMsg), "upstream_error", resp.StatusCode)
				return
			}
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("body: %s", string(responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("job id is empty, response: %s", string(responseBody)), "invalid_response", http.StatusInternalServerError)
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

	url := baseUrl + "/v1/jobs/" + taskID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var dResp jobResponse
	if err := common.Unmarshal(respBody, &dResp); err != nil {
		return nil, fmt.Errorf("unmarshal job response failed: %w", err)
	}

	taskInfo := &relaycommon.TaskInfo{
		Code:   0,
		TaskID: dResp.ID,
	}

	switch dResp.Status {
	case "completed", "succeeded":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		// Extract video URL from response
		if dResp.URL != "" {
			taskInfo.Url = dResp.URL
		} else if len(dResp.URLs) > 0 {
			taskInfo.Url = dResp.URLs[0]
		} else if len(dResp.Choices) > 0 {
			// Extract URL from HTML content
			content := dResp.Choices[0].Message.Content
			if strings.Contains(content, "<video") {
				// Extract src from <video src='...'>
				if idx := strings.Index(content, "src='"); idx >= 0 {
					start := idx + 5
					if end := strings.Index(content[start:], "'"); end >= 0 {
						taskInfo.Url = content[start : start+end]
					}
				} else if idx := strings.Index(content, "src=\""); idx >= 0 {
					start := idx + 5
					if end := strings.Index(content[start:], "\""); end >= 0 {
						taskInfo.Url = content[start : start+end]
					}
				}
			}
		}
	case "failed", "failure":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		if dResp.Error != nil && dResp.Error.Message != "" {
			taskInfo.Reason = dResp.Error.Message
		}
	case "queued":
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressQueued
	case "processing":
		taskInfo.Status = model.TaskStatusInProgress
		if dResp.Progress > 0 {
			taskInfo.Progress = fmt.Sprintf("%d%%", dResp.Progress)
		} else {
			taskInfo.Progress = taskcommon.ProgressInProgress
		}
	default:
		taskInfo.Status = model.TaskStatusInProgress
		taskInfo.Progress = taskcommon.ProgressInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var dResp jobResponse
	if err := common.Unmarshal(task.Data, &dResp); err != nil {
		return nil, fmt.Errorf("unmarshal secureskill task data failed: %w", err)
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	openAIVideo.CreatedAt = task.CreatedAt
	openAIVideo.CompletedAt = task.UpdatedAt

	// Get URL from task result
	url := task.GetResultURL()
	if url == "" {
		// Try to extract from task data
		if dResp.URL != "" {
			url = dResp.URL
		} else if len(dResp.URLs) > 0 {
			url = dResp.URLs[0]
		}
	}
	if url != "" {
		openAIVideo.SetMetadata("url", url)
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
		"omni",
		"omni_video_edit",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "SecureSkill"
}
