package zhangyuge

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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

	// Compress base64 images if they exceed 10MB
	bodyMap = service.CompressImageInBodyMap(bodyMap, 10<<20)

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
		size = strings.TrimSpace(v)
	}
	aspectRatio := ""
	if v, ok := bodyMap["aspect_ratio"].(string); ok {
		aspectRatio = strings.TrimSpace(v)
	}
	// Also check "resolution" field used by some clients
	if size == "" {
		if v, ok := bodyMap["resolution"].(string); ok {
			size = strings.TrimSpace(v)
		}
	}
	common.SysLog(fmt.Sprintf("[ZhangyugeAI] raw size=%q aspect_ratio=%q model=%q", size, aspectRatio, model))

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

	// Distinguish request format by model type
	isGPTImage := strings.HasPrefix(model, "gpt-image-2")
	if isGPTImage {
		// gpt-image-2 uses metadata.aspect_ratio and metadata.urls
		metadata := make(map[string]interface{})

		// aspect_ratio: prefer explicit aspect_ratio, fallback to size mapping
		ar := aspectRatio
		if ar == "" && size != "" {
			ar = mapSizeToAspectRatio(size)
		}
		if ar != "" {
			metadata["aspect_ratio"] = ar
		}

		// urls: reference images (max 5 for gpt-image-2)
		if len(images) > 0 {
			if len(images) > 5 {
				common.SysLog(fmt.Sprintf("[ZhangyugeAI] warning: %d images provided for gpt-image-2, truncating to 5", len(images)))
				images = images[:5]
			}
			metadata["urls"] = images
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] sending %d reference images for gpt-image-2", len(images)))
		}

		if len(metadata) > 0 {
			zhangyugeBody["metadata"] = metadata
		}
	} else {
		// omni_flash-10s / veo use size and images
		if size != "" {
			zhangyugeBody["size"] = mapSizeToZhangyuge(size, aspectRatio)
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] mapped size=%q", zhangyugeBody["size"]))
		}
		if len(images) > 0 {
			// omni_flash-10s supports up to 7 reference images
			if len(images) > 7 {
				common.SysLog(fmt.Sprintf("[ZhangyugeAI] warning: %d images provided, truncating to 7", len(images)))
				images = images[:7]
			}
			zhangyugeBody["images"] = images
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] sending %d images", len(images)))
		}
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
	Error       any    `json:"error"`
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
		if dResp.Error != nil {
			switch v := dResp.Error.(type) {
			case string:
				taskInfo.Reason = v
			case map[string]any:
				if msg, ok := v["message"].(string); ok {
					taskInfo.Reason = msg
				} else {
					taskInfo.Reason = fmt.Sprintf("%v", v)
				}
			default:
				taskInfo.Reason = fmt.Sprintf("%v", v)
			}
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

	if zResp.VideoURL != "" {
		openAIVideo.SetMetadata("url", zResp.VideoURL)
	} else if zResp.URL != "" {
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
	case "omni-flash-10s", "omni_flash_10s":
		return "omni_flash-10s"
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
	// Handle aspect-ratio sent directly as size (common in some client integrations)
	case "9:16":
		return "1080x1920"
	case "16:9":
		return "1920x1080"
	case "1:1":
		return "1080x1080"
	default:
		return size
	}
}

// mapSizeToAspectRatio converts widthxheight to aspect ratio string.
// Supports common sizes used by GPT image generation APIs.
func mapSizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024", "1080x1080", "512x512", "768x768":
		return "1:1"
	case "1024x1536", "1080x1920", "720x1280", "768x1344", "576x1024":
		return "9:16"
	case "1920x1080", "1280x720", "1344x768", "1024x576":
		return "16:9"
	case "1024x1280", "768x960":
		return "4:5"
	case "1280x1024", "960x768":
		return "5:4"
	case "1024x768":
		return "4:3"
	case "768x1024":
		return "3:4"
	case "1536x1024", "1152x768":
		return "3:2"
	case "768x1152":
		return "2:3"
	case "1920x823", "2560x1080":
		return "21:9"
	default:
		// Try to parse widthxheight and compute ratio
		parts := strings.Split(size, "x")
		if len(parts) == 2 {
			w, err1 := strconv.Atoi(parts[0])
			h, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && h > 0 {
				// Simplify ratio by GCD
				g := gcd(w, h)
				if g > 0 {
					return fmt.Sprintf("%d:%d", w/g, h/g)
				}
			}
		}
		return "1:1" // default
	}
}

func gcd(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

// parseMultipartBody parses multipart/form-data into a map similar to JSON body.
// Text fields are extracted directly. File fields (images) are handled as follows:
//   - If content looks like a URL or base64 data URI, use it directly.
//   - If binary image data, encode as base64 data URI.
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
			if len(images) >= 7 {
				common.SysLog(fmt.Sprintf("[ZhangyugeAI] multipart: skipping image #%d, max 7 allowed", len(images)+1))
				continue
			}
			imgURL := handleImagePart(c, data, part.Header.Get("Content-Type"))
			if imgURL != "" {
				images = append(images, imgURL)
			}
			continue
		}

		// Text parts — trim trailing CRLF/LF from multipart form values
		result[name] = strings.TrimSpace(string(data))
	}

	if len(images) > 0 {
		result["images"] = images
		common.SysLog(fmt.Sprintf("[ZhangyugeAI] multipart: collected %d images", len(images)))
	}
	return result
}

func isImageFieldName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "image" || lower == "images" || strings.HasPrefix(lower, "input_reference")
}

// handleImagePart processes an image part. If it looks like a URL or base64, returns as-is.
// Binary image data is encoded as base64 data URI so Zhangyuge servers can access it directly.
func handleImagePart(c *gin.Context, data []byte, contentType string) string {
	str := string(data)
	str = strings.TrimSpace(str)

	// Direct URL
	if strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") {
		return str
	}

	// Base64 data URI — pass through
	if strings.HasPrefix(str, "data:") {
		return str
	}

	// Binary image data
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		// Not an image, try treating as URL string anyway
		return str
	}

	// Compress if binary image exceeds 10MB
	if len(data) > service.DefaultMaxImageBytes {
		compressed, _, err := service.CompressImageBytes(data, contentType, service.DefaultMaxImageBytes)
		if err == nil {
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] compressed multipart image: before=%d bytes, after=%d bytes", len(data), len(compressed)))
			data = compressed
			contentType = "image/jpeg"
		} else {
			common.SysLog(fmt.Sprintf("[ZhangyugeAI] failed to compress multipart image (%d bytes): %v", len(data), err))
			return "" // Reject oversized images that cannot be compressed
		}
	}

	// Encode binary image as base64 data URI — Zhangyuge accepts data URIs directly
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, b64)
	common.SysLog(fmt.Sprintf("[ZhangyugeAI] encoded multipart image to data URI: %s, %d bytes", contentType, len(data)))
	return dataURI
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"omni_flash-10s",
		"veo_3_1-fast",
		"veo_3_1",
		"gpt-image-2",
		"gpt-image-2-2K",
		"gpt-image-2-4K",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "zhangyuge"
}
