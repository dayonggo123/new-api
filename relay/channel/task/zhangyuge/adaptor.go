package zhangyuge

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// TaskAdaptor implements the task platform interface for ZhangyugeAI (章鱼哥AI).
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
		// Try parsing as multipart/form-data (e.g. OpenAI Video API with image uploads)
		contentType := c.Request.Header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			bodyMap = parseMultipartBody(c, cachedBody, contentType)
		}
		if len(bodyMap) == 0 {
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] JSON unmarshal failed, content-type=%s, returning raw body", contentType))
			return bytes.NewReader(cachedBody), nil
		}
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

	zhangyugeBody := make(map[string]interface{})

	// model: prefer info.UpstreamModelName (already processed by ModelMappedHelper),
	// fallback to bodyMap["model"] for direct API calls without channel mapping
	model := info.UpstreamModelName
	if model == "" {
		if m, ok := bodyMap["model"].(string); ok && m != "" {
			model = m
		}
	}
	if model != "" {
		zhangyugeBody["model"] = mapModelName(model)
	}

	// prompt
	if prompt, ok := bodyMap["prompt"].(string); ok && prompt != "" {
		zhangyugeBody["prompt"] = prompt
	}

	// size: map 1080p/720p to widthxheight, respecting aspect_ratio
	size := ""
	if v, ok := bodyMap["size"].(string); ok {
		size = v
	}
	aspectRatio := ""
	if v, ok := bodyMap["aspect_ratio"].(string); ok {
		aspectRatio = v
	}
	if size != "" {
		zhangyugeBody["size"] = mapSizeToZhangyuge(size, aspectRatio)
	}

	// images: support both "images" (array) and "image" (single string)
	var images []interface{}
	if imgs, ok := bodyMap["images"]; ok {
		if imgList, ok := imgs.([]interface{}); ok {
			images = append(images, imgList...)
		} else if imgStr, ok := imgs.(string); ok && imgStr != "" {
			images = append(images, imgStr)
		}
	}
	if img, ok := bodyMap["image"].(string); ok && img != "" {
		images = append(images, img)
	}
	if len(images) > 0 {
		zhangyugeBody["images"] = images
	}

	jsonData, err := common.Marshal(zhangyugeBody)
	if err != nil {
		return nil, err
	}
	common.SysLog(fmt.Sprintf("[ZhangyugeAI] upstream request body: %s", string(jsonData)))
	return bytes.NewReader(jsonData), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

type submitResponse struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CreatedAt   int64  `json:"created_at"`
	CompletedAt int64  `json:"completed_at"`
	URL         string `json:"url"`
	VideoURL    string `json:"video_url"`
	Size        string `json:"size"`
	Error       string `json:"error"`
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
		if dResp.VideoURL != "" {
			taskInfo.Url = dResp.VideoURL
		} else {
			taskInfo.Url = dResp.URL
		}
	case "failed", "failure":
		taskInfo.Status = model.TaskStatusFailure
		taskInfo.Progress = "100%"
		taskInfo.Reason = dResp.Error
	default:
		taskInfo.Status = model.TaskStatusInProgress
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var zResp submitResponse
	if err := common.Unmarshal(task.Data, &zResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal zhangyuge task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = task.TaskID
	openAIVideo.Status = task.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(task.Progress)
	openAIVideo.CreatedAt = task.CreatedAt
	openAIVideo.CompletedAt = task.UpdatedAt

	if zResp.URL != "" {
		openAIVideo.SetMetadata("url", zResp.URL)
	}

	if task.Status == model.TaskStatusFailure && task.FailReason != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: task.FailReason,
			Code:    "task_failed",
		}
	}

	return common.Marshal(openAIVideo)
}

func mapModelName(model string) string {
	// Strip provider prefix if any
	if idx := strings.Index(model, "/"); idx >= 0 {
		model = model[idx+1:]
	}
	switch model {
	case "veo-3.1-fast", "veo_3_1-fast":
		return "veo_3_1"
	case "veo-3.1-fast-fl", "veo_3_1-fast-fl":
		return "veo_3_1-fast-fl"
	default:
		return model
	}
}

func mapSizeToZhangyuge(size string, aspectRatio string) string {
	switch size {
	case "1080p":
		if aspectRatio == "9:16" {
			return "1080x1920"
		}
		return "1920x1080"
	case "720p":
		if aspectRatio == "9:16" {
			return "720x1280"
		}
		return "1280x720"
	default:
		return size
	}
}

// parseMultipartBody parses multipart/form-data into a map similar to JSON body.
// Text fields are extracted directly. File fields (images) are handled as follows:
//   - If content looks like a URL or base64 data URI, use it directly.
//   - If binary image data, save to uploads/ and generate a public URL.
func parseMultipartBody(c *gin.Context, body []byte, contentType string) map[string]interface{} {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil
	}
	boundary, ok := params["boundary"]
	if !ok || boundary == "" {
		return nil
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	result := make(map[string]interface{})
	var images []interface{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		name := part.FormName()
		data, err := io.ReadAll(part)
		if err != nil {
			continue
		}
		if len(data) == 0 {
			continue
		}

		// File parts (images)
		if part.FileName() != "" || isImageFieldName(name) {
			imgURL := handleImagePart(c, data, part.Header.Get("Content-Type"))
			if imgURL != "" {
				images = append(images, imgURL)
			}
			continue
		}

		// Text parts
		result[name] = string(data)
	}

	if len(images) > 0 {
		result["images"] = images
	}
	return result
}

func isImageFieldName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "image" || lower == "images" || lower == "input_reference" || lower == "input_references"
}

// handleImagePart processes an image part. If it looks like a URL or base64, returns as-is.
// Otherwise saves binary data to uploads/ and returns a public URL.
func handleImagePart(c *gin.Context, data []byte, contentType string) string {
	str := string(data)
	str = strings.TrimSpace(str)

	// Direct URL
	if strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") {
		return str
	}

	// Base64 data URI
	if strings.HasPrefix(str, "data:") {
		return str
	}

	// Binary image data: save to uploads/
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		// Not an image, try treating as URL string anyway
		return str
	}

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		common.SysLog(fmt.Sprintf("[ZhangyugeAI] failed to create upload dir: %v", err))
		return ""
	}

	ext := "bin"
	switch contentType {
	case "image/png":
		ext = "png"
	case "image/jpeg", "image/jpg":
		ext = "jpg"
	case "image/gif":
		ext = "gif"
	case "image/webp":
		ext = "webp"
	}

	filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)
	filePath := filepath.Join(uploadDir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		common.SysLog(fmt.Sprintf("[ZhangyugeAI] failed to write upload file: %v", err))
		return ""
	}

	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/uploads/%s", scheme, host, filename)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"default"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "zhangyuge"
}
