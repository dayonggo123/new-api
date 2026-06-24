package apimart

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// apimartLog writes debug logs to a fixed file so they survive Gin ReleaseMode.
func apimartLog(s string) {
	f, err := os.OpenFile("/tmp/apimart_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006/01/02 15:04:05"), s)
}

var base64Pattern = regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)

func getBaseURL(c *gin.Context) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}

func extFromContentType(ct string) string {
	switch strings.ToLower(ct) {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	case "image/heic":
		return "heic"
	case "image/heif":
		return "heif"
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	default:
		return "bin"
	}
}

func saveTempUpload(data []byte, ext string) (string, error) {
	dir := "./uploads"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s.%s", uuid.New().String(), ext)
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	return filename, nil
}

func looksLikeBase64(s string) bool {
	if len(s) < 100 {
		return false
	}
	return base64Pattern.MatchString(s)
}

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
	contentType := c.GetHeader("Content-Type")

	// 下游改为 multipart/form-data + 二进制文件上传时的处理路径
	if strings.Contains(contentType, "multipart/form-data") {
		req, err := a.parseMultipartToTaskSubmitReq(c, getBaseURL(c))
		if err != nil {
			return nil, err
		}
		common.SysLog(fmt.Sprintf("[APIMart] multipart req: prompt=%q model=%q size=%q aspect_ratio=%q refImages=%d images=%d imageURLs=%d videoURLs=%d image=%q metadata=%v",
			req.Prompt, req.Model, req.Size, req.AspectRatio, len(req.ReferenceImages), len(req.Images), len(req.ImageURLs), len(req.VideoURLs), req.Image, req.Metadata))
		apimartLog(fmt.Sprintf("[APIMart] BuildRequestBody (multipart): prompt=%q size=%q aspect_ratio=%q referenceImages=%d images=%d imageURLs=%d image=%q metadata=%v",
			req.Prompt, req.Size, req.AspectRatio, len(req.ReferenceImages), len(req.Images), len(req.ImageURLs), req.Image, req.Metadata))
		body, err := a.convertToRequestPayload(req, info)
		if err != nil {
			return nil, err
		}
		apimartLog(fmt.Sprintf("[APIMart] request body: %s", string(body)))
		c.Set("apimart_request_body", string(body))
		return bytes.NewReader(body), nil
	}

	// 原有的 JSON 处理路径
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	common.SysLog(fmt.Sprintf("[APIMart] json req: prompt=%q model=%q size=%q aspect_ratio=%q refImages=%d images=%d imageURLs=%d videoURLs=%d image=%q metadata=%v",
		req.Prompt, req.Model, req.Size, req.AspectRatio, len(req.ReferenceImages), len(req.Images), len(req.ImageURLs), len(req.VideoURLs), req.Image, req.Metadata))
	apimartLog(fmt.Sprintf("[APIMart] BuildRequestBody: prompt=%q size=%q aspect_ratio=%q images=%d imageURLs=%d referenceImages=%d metadata=%v", req.Prompt, req.Size, req.AspectRatio, len(req.Images), len(req.ImageURLs), len(req.ReferenceImages), req.Metadata))

	body, err := a.convertToRequestPayload(req, info)
	if err != nil {
		return nil, err
	}
	apimartLog(fmt.Sprintf("[APIMart] request body: %s", string(body)))
	c.Set("apimart_request_body", string(body))
	return bytes.NewReader(body), nil
}

// parseMultipartToTaskSubmitReq 从 multipart/form-data 请求中解析出 TaskSubmitReq。
// 兼容下游 ewapi/client.rs 的新逻辑：文本字段 + 文件字段（ref_images）上传参考图。
func (a *TaskAdaptor) parseMultipartToTaskSubmitReq(c *gin.Context, baseURL string) (relaycommon.TaskSubmitReq, error) {
	formData, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return relaycommon.TaskSubmitReq{}, err
	}
	defer formData.RemoveAll()

	req := relaycommon.TaskSubmitReq{
		Metadata: make(map[string]interface{}),
	}

	// ----- 文本字段 -----
	if v, ok := formData.Value["prompt"]; ok && len(v) > 0 {
		req.Prompt = v[0]
	}
	if v, ok := formData.Value["model"]; ok && len(v) > 0 {
		req.Model = v[0]
	}
	if v, ok := formData.Value["size"]; ok && len(v) > 0 {
		req.Size = v[0]
	}
	if v, ok := formData.Value["aspect_ratio"]; ok && len(v) > 0 {
		req.AspectRatio = v[0]
	}
	if v, ok := formData.Value["image"]; ok && len(v) > 0 {
		req.Image = v[0]
	}
	if v, ok := formData.Value["duration"]; ok && len(v) > 0 {
		if d, err := strconv.Atoi(v[0]); err == nil {
			req.Duration = d
		}
	}
	if v, ok := formData.Value["seconds"]; ok && len(v) > 0 {
		if d, err := strconv.Atoi(v[0]); err == nil {
			req.Duration = d
		}
	}

	// 文本形式的图片 URL / base64（向下兼容）
	if images, ok := formData.Value["images"]; ok {
		req.Images = images
	}
	if refImages, ok := formData.Value["reference_images"]; ok {
		req.ReferenceImages = refImages
	}
	if imageURLs, ok := formData.Value["image_urls"]; ok {
		req.ImageURLs = imageURLs
	}
	if videoURLs, ok := formData.Value["video_urls"]; ok {
		req.VideoURLs = videoURLs
	}

	// ----- 文件字段（二进制图片）-----
	// 下游 ewapi 使用 ref_images 作为文件字段名；同时兼容 images / files
	fileFieldNames := []string{"ref_images", "images", "files"}
	for _, fieldName := range fileFieldNames {
		fileHeaders, ok := formData.File[fieldName]
		if !ok {
			continue
		}
		for _, fh := range fileHeaders {
			f, err := fh.Open()
			if err != nil {
				apimartLog(fmt.Sprintf("[APIMart] failed to open multipart file %s: %v", fh.Filename, err))
				continue
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				apimartLog(fmt.Sprintf("[APIMart] failed to read multipart file %s: %v", fh.Filename, err))
				continue
			}
			if len(data) == 0 {
				continue
			}

			ct := fh.Header.Get("Content-Type")
			if ct == "" || ct == "application/octet-stream" {
				ct = http.DetectContentType(data)
			}
			ext := extFromContentType(ct)
			filename, err := saveTempUpload(data, ext)
			if err != nil {
				common.SysLog(fmt.Sprintf("[APIMart] failed to save upload file %s: %v", fh.Filename, err))
				apimartLog(fmt.Sprintf("[APIMart] failed to save upload file %s: %v", fh.Filename, err))
				continue
			}
			url := baseURL + "/uploads/" + filename
			req.ReferenceImages = append(req.ReferenceImages, url)
			common.SysLog(fmt.Sprintf("[APIMart] multipart file %s -> local URL %s (%s, %d bytes)", fh.Filename, url, ct, len(data)))
			apimartLog(fmt.Sprintf("[APIMart] multipart file %s -> local URL %s (%s, %d bytes)", fh.Filename, url, ct, len(data)))
		}
	}

	// ----- Metadata：收集未知字段 -----
	knownFields := map[string]bool{
		"prompt": true, "model": true, "size": true, "aspect_ratio": true,
		"image": true, "images": true, "reference_images": true,
		"video_urls": true, "duration": true, "seconds": true, "mode": true,
		"ref_images": true, "files": true,
	}
	for key, values := range formData.Value {
		if knownFields[key] || len(values) == 0 {
			continue
		}
		valStr := values[0]
		if intVal, err := strconv.Atoi(valStr); err == nil {
			req.Metadata[key] = intVal
		} else if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
			req.Metadata[key] = floatVal
		} else if boolVal, err := strconv.ParseBool(valStr); err == nil {
			req.Metadata[key] = boolVal
		} else {
			req.Metadata[key] = valStr
		}
	}

	return req, nil
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

	// Collect image URLs - 图生图场景优先使用 ReferenceImages，向下兼容 Images / ImageURLs / Image
	imageURLs := req.ReferenceImages
	if len(imageURLs) == 0 {
		imageURLs = req.Images
	}
	if len(imageURLs) == 0 {
		imageURLs = req.ImageURLs
	}
	if len(imageURLs) == 0 && req.Image != "" {
		imageURLs = []string{req.Image}
	}

	// APIMart 接受 http/https/asset:// URL 和 data: base64 URI
	var validURLs []string
	var droppedURLs []string
	for _, url := range imageURLs {
		url = strings.TrimSpace(url)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "asset://") {
			validURLs = append(validURLs, url)
		} else if strings.HasPrefix(url, "data:") {
			// data: URI 直接透传给上游（章鱼哥上游原生支持）
			validURLs = append(validURLs, url)
		} else {
			droppedURLs = append(droppedURLs, url[:min(len(url), 80)])
		}
	}
	imageURLs = validURLs

	if len(droppedURLs) > 0 {
		common.SysLog(fmt.Sprintf("[APIMart] dropped %d unsupported URLs (only http/https/asset:// allowed): %v", len(droppedURLs), droppedURLs))
	}

	apimartLog(fmt.Sprintf("[APIMart] image source: referenceImages=%d images=%d imageURLs=%d image=%q final=%d dropped=%d",
		len(req.ReferenceImages), len(req.Images), len(req.ImageURLs), req.Image, len(imageURLs), len(droppedURLs)))

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
		payload := map[string]interface{}{
			"model":   info.UpstreamModelName,
			"prompt":  req.Prompt,
			"n":       1,
		}

		// 对齐 APIMart 官方格式：
		//   size 字段传比例字符串（如 "16:9"）或像素尺寸（如 "1920x1080"）
		//   resolution 字段传档位（"1k" / "2k" / "4k"）
		// 优先用 aspectRatio（已收集到的比例），其次用 req.Size
		if aspectRatio != "" {
			payload["size"] = aspectRatio
		} else if req.Size != "" {
			payload["size"] = req.Size
		} else {
			payload["size"] = "1:1"
		}

		if len(imageURLs) > 0 {
			payload["image_urls"] = imageURLs
		}

		// resolution：默认 1k
		if req.Metadata != nil {
			if v, ok := req.Metadata["resolution"].(string); ok && v != "" {
				payload["resolution"] = v
			} else {
				payload["resolution"] = "1k"
			}
			if v, ok := req.Metadata["n"].(float64); ok {
				payload["n"] = int(v)
			}
			if v, ok := req.Metadata["official_fallback"].(bool); ok {
				payload["official_fallback"] = v
			}
		} else {
			payload["resolution"] = "1k"
		}

		body, _ := common.Marshal(payload)
		apimartLog(fmt.Sprintf("[APIMart] image request payload: %s", string(body)))
		return body, nil
	}

	// Video generation
	payload := map[string]interface{}{
		"model":  info.UpstreamModelName,
		"prompt": req.Prompt,
	}

	hasVideoURLs := len(req.VideoURLs) > 0

	// Duration: default 6s; mutually exclusive with video_urls
	if hasVideoURLs {
		payload["video_urls"] = req.VideoURLs
	} else if req.Duration > 0 {
		payload["duration"] = req.Duration
	} else {
		payload["duration"] = 6
	}

	if len(imageURLs) > 0 {
		payload["image_urls"] = imageURLs
	}

	// aspect_ratio from collected value, req.Size (if ratio format), or default
	if aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	} else if req.Size != "" {
		if strings.Contains(req.Size, ":") {
			payload["aspect_ratio"] = req.Size
		} else {
			payload["aspect_ratio"] = mapSizeToAspectRatio(req.Size)
		}
	} else {
		payload["aspect_ratio"] = "16:9"
	}

	// resolution: default 720p
	resolution := "720p"
	if req.Metadata != nil {
		if v, ok := req.Metadata["resolution"].(string); ok && v != "" {
			resolution = strings.ToLower(v)
		}
		if v, ok := req.Metadata["generation_type"].(string); ok && v != "" {
			payload["generation_type"] = v
		} else if len(imageURLs) > 0 {
			// Omni-Flash-Ext: 1 image = single image, 3 images = reference fusion
			if len(imageURLs) >= 3 {
				payload["generation_type"] = "reference"
			}
		}
		if v, ok := req.Metadata["enable_gif"].(bool); ok {
			payload["enable_gif"] = v
		}
		if v, ok := req.Metadata["official_fallback"].(bool); ok {
			payload["official_fallback"] = v
		}
	} else if req.Size != "" {
		// Allow size like "720p", "1080p", "4k" to be used as resolution
		lowerSize := strings.ToLower(req.Size)
		if lowerSize == "720p" || lowerSize == "1080p" || lowerSize == "4k" {
			resolution = lowerSize
		}
	}
	payload["resolution"] = resolution

	apimartLog(fmt.Sprintf("[APIMart] video payload: model=%s prompt=%q duration=%d aspect_ratio=%s resolution=%s videoURLs=%d imageURLs=%d",
		info.UpstreamModelName, req.Prompt, req.Duration, payload["aspect_ratio"], resolution, len(req.VideoURLs), len(imageURLs)))

	body, _ := common.Marshal(payload)
	apimartLog(fmt.Sprintf("[APIMart] video request payload: %s", string(body)))
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

	apimartLog(fmt.Sprintf("[APIMart] create response body: %s", string(responseBody)))

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
		"Omni-Flash-Ext",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "apimart"
}
