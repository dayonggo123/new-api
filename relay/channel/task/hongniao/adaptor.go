package hongniao

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

// TaskAdaptor implements the task-based channel for HongniaoAI (红鸟 AI).
// It supports both asynchronous image generation (POST /v1/images) and video
// generation (POST /v1/videos) in the OpenAI-compatible format.
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
	action := constant.TaskActionGenerate
	if a.isImageRequest(info) {
		action = constant.TaskActionImageGenerate
	}
	if err := relaycommon.ValidateBasicTaskRequest(c, info, action); err != nil {
		return err
	}
	return nil
}

// isImageRequest determines whether the current request is an image generation
// request by inspecting the downstream request path. This allows users to
// configure any model names in the backend without hardcoding image/video rules.
func (a *TaskAdaptor) isImageRequest(info *relaycommon.RelayInfo) bool {
	return strings.Contains(info.RequestURLPath, "/images/")
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.isImageRequest(info) {
		return fmt.Sprintf("%s/v1/images", a.baseURL), nil
	}
	return fmt.Sprintf("%s/v1/videos", a.baseURL), nil
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

	common.SysLog(fmt.Sprintf("[HONGNIAO] TaskSubmitReq: images=%d, imageURLs=%d, videoURLs=%d, refVideos=%d, refAudio=%d, image=%q",
		len(req.Images), len(req.ImageURLs), len(req.VideoURLs), len(req.ReferenceVideo), len(req.ReferenceAudio), req.Image))

	isImage := a.isImageRequest(info)

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
	if len(images) > 0 {
		payload["images"] = images
	}

	if isImage {
		// Image generation fields.
		if req.Size != "" {
			payload["size"] = req.Size
		}
		if req.Resolution != "" {
			payload["resolution"] = req.Resolution
		}
		// aspect_ratio for images
		aspectRatio := req.AspectRatio
		if aspectRatio == "" && req.Ratio != "" {
			aspectRatio = req.Ratio
		}
		if aspectRatio == "" && req.Size != "" {
			if strings.Contains(req.Size, ":") {
				aspectRatio = req.Size
			} else {
				aspectRatio, _ = taskcommon.ParseSizeToRatio(req.Size)
			}
		}
		if aspectRatio == "" && req.Metadata != nil {
			if raw, ok := req.Metadata["aspect_ratio"]; ok {
				if s, ok := raw.(string); ok {
					aspectRatio = s
				}
			}
			if aspectRatio == "" {
				if raw, ok := req.Metadata["aspectRatio"]; ok {
					if s, ok := raw.(string); ok {
						aspectRatio = s
					}
				}
			}
		}
		if aspectRatio != "" {
			payload["aspect_ratio"] = aspectRatio
		}
	} else {
		// Video generation fields.
		var videos []string
		if len(req.VideoURLs) > 0 {
			videos = req.VideoURLs
		} else if len(req.ReferenceVideo) > 0 {
			videos = req.ReferenceVideo
		}
		if len(videos) > 0 {
			// Hongniao upstream expects camelCase videoUrls / audioUrls.
			payload["videoUrls"] = videos
			payload["videos"] = videos
		}

		var audios []string
		if len(req.ReferenceAudio) > 0 {
			audios = req.ReferenceAudio
		}
		if len(audios) > 0 {
			payload["audioUrls"] = audios
			payload["audios"] = audios
		}

		// aspectRatio: prefer TaskSubmitReq.AspectRatio, then Ratio, then Size, then metadata
		// DEBUG: log the values
		fmt.Printf("[Hongniao] req.AspectRatio=%q req.Ratio=%q req.Size=%q\n", req.AspectRatio, req.Ratio, req.Size)
		aspectRatio := req.AspectRatio
		if aspectRatio == "" && req.Ratio != "" {
			aspectRatio = req.Ratio
		}
		if aspectRatio == "" && req.Size != "" {
			if strings.Contains(req.Size, ":") {
				aspectRatio = req.Size
			} else {
				aspectRatio, _ = taskcommon.ParseSizeToRatio(req.Size)
			}
		}
		if aspectRatio == "" && req.Metadata != nil {
			if raw, ok := req.Metadata["aspectRatio"]; ok {
				if s, ok := raw.(string); ok {
					aspectRatio = s
				}
			}
			if aspectRatio == "" {
				if raw, ok := req.Metadata["aspect_ratio"]; ok {
					if s, ok := raw.(string); ok {
						aspectRatio = s
					}
				}
			}
		}
		if aspectRatio == "" {
			aspectRatio = "16:9"
		}
		payload["aspect_ratio"] = aspectRatio

		// seconds: prefer TaskSubmitReq.Seconds, then Duration, then metadata
		seconds := req.Seconds
		if seconds == "" && req.Duration > 0 {
			seconds = strconv.Itoa(req.Duration)
		}
		if seconds == "" && req.Metadata != nil {
			if raw, ok := req.Metadata["seconds"]; ok {
				switch v := raw.(type) {
				case string:
					seconds = v
				case int:
					seconds = strconv.Itoa(v)
				case float64:
					seconds = strconv.Itoa(int(v))
				}
			}
		}
		if seconds == "" {
			seconds = "10"
		}
		payload["seconds"] = seconds

		// resolution
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
	}

	// Merge remaining metadata fields that are not already handled.
	knownFields := map[string]bool{
		"model": true, "prompt": true, "aspectRatio": true, "aspect_ratio": true,
		"seconds": true, "duration": true, "images": true, "videos": true,
		"audios": true, "videoUrls": true, "audioUrls": true,
		"image": true, "image_urls": true, "reference_images": true,
		"video_urls": true, "reference_video": true, "reference_audio": true,
		"size": true, "ratio": true, "resolution": true, "n": true,
		"parameters": true, "metadata": true,
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
		return nil, errors.Wrap(err, "marshal hongniao request body failed")
	}
	common.SysLog(fmt.Sprintf("[HONGNIAO] request body: %s", string(body)))
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// submitResponse matches Hongniao's POST /v1/videos and /v1/images response.
type submitResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object,omitempty"`
	Model     string `json:"model,omitempty"`
	Status    string `json:"status,omitempty"`
	Progress  int    `json:"progress,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Size      string `json:"size,omitempty"`
	Seconds   string `json:"seconds,omitempty"`
}

// queryResponse matches Hongniao's GET /v1/videos/{id} and /v1/images/{id} response.
type queryResponse struct {
	ID        string       `json:"id"`
	Object    string       `json:"object,omitempty"`
	Model     string       `json:"model,omitempty"`
	Status    string       `json:"status,omitempty"`
	Progress  int          `json:"progress,omitempty"`
	CreatedAt int64        `json:"created_at,omitempty"`
	Size      string       `json:"size,omitempty"`
	Seconds   string       `json:"seconds,omitempty"`
	VideoURL  string       `json:"video_url,omitempty"`
	URL       string       `json:"url,omitempty"`
	Images    []string     `json:"images,omitempty"`
	Result    *queryResult `json:"result,omitempty"`
	Error     *queryError  `json:"error,omitempty"`
}

type queryResult struct {
	VideoURL string   `json:"video_url,omitempty"`
	ImageURL string   `json:"image_url,omitempty"`
	URL      string   `json:"url,omitempty"`
	Images   []string `json:"images,omitempty"`
	Videos   []string `json:"videos,omitempty"`
}

type queryError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
	Type    string `json:"type,omitempty"`
}

// hongniaoErrorResponse matches Hongniao's error payload.
// Example: {"code":400,"message":"...","error":"Bad Request"}
type hongniaoErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
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

	common.SysLog(fmt.Sprintf("[HONGNIAO] submit response body: %s", string(responseBody)))

	// Hongniao returns error payloads as {"code":400,"message":"...","error":"..."}
	var errResp hongniaoErrorResponse
	if err := common.Unmarshal(responseBody, &errResp); err == nil && errResp.Code != 0 && errResp.Code != 200 {
		msg := errResp.Message
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = fmt.Sprintf("hongniao returned code %d", errResp.Code)
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("%s", msg), "hongniao_error", errResp.Code)
		return
	}

	var sResp submitResponse
	if err := common.Unmarshal(responseBody, &sResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if sResp.ID == "" {
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

	return sResp.ID, responseBody, nil
}

// isImageTaskFromBody checks whether the stored task was created from an image
// request. The polling layer passes task.Action in the body map.
func (a *TaskAdaptor) isImageTaskFromBody(body map[string]any) bool {
	if action, ok := body["action"].(string); ok {
		return action == constant.TaskActionImageGenerate
	}
	return false
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	endpoint := "videos"
	if a.isImageTaskFromBody(body) {
		endpoint = "images"
	}
	url := fmt.Sprintf("%s/v1/%s/%s", baseUrl, endpoint, taskID)
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

	if qResp.Progress > 0 {
		taskInfo.Progress = fmt.Sprintf("%d%%", qResp.Progress)
	}

	status := strings.ToLower(qResp.Status)
	switch status {
	case "completed":
		taskInfo.Status = model.TaskStatusSuccess
		taskInfo.Progress = "100%"
		// Extract result URL for both video and image tasks.
		if qResp.URL != "" {
			taskInfo.Url = qResp.URL
		} else if qResp.VideoURL != "" {
			taskInfo.Url = qResp.VideoURL
		} else if len(qResp.Images) > 0 && qResp.Images[0] != "" {
			taskInfo.Url = qResp.Images[0]
		} else if qResp.Result != nil {
			if qResp.Result.URL != "" {
				taskInfo.Url = qResp.Result.URL
			} else if qResp.Result.VideoURL != "" {
				taskInfo.Url = qResp.Result.VideoURL
			} else if qResp.Result.ImageURL != "" {
				taskInfo.Url = qResp.Result.ImageURL
			} else if len(qResp.Result.Images) > 0 && qResp.Result.Images[0] != "" {
				taskInfo.Url = qResp.Result.Images[0]
			} else if len(qResp.Result.Videos) > 0 && qResp.Result.Videos[0] != "" {
				taskInfo.Url = qResp.Result.Videos[0]
			}
		}
	case "failed", "failure":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		if qResp.Error != nil && qResp.Error.Message != "" {
			taskInfo.Reason = qResp.Error.Message
		} else {
			taskInfo.Reason = qResp.Status
		}
	case "queued", "processing":
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

// GetModelList returns an empty list because model names and prices are
// configured by the user in the backend channel settings.
func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		// Video models
		"keling-3",
		"sdquan-2",
		"video-2.0-fast-720P",
		// Image models (examples; add real upstream image model IDs as needed)
		"gpt-image2",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "hongniao"
}
