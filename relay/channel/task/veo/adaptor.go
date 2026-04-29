package veo

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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

type submitResponse struct {
	ID                 int     `json:"id"`
	UUID               string  `json:"uuid"`
	UserID             int     `json:"user_id"`
	ModelName          string  `json:"model_name"`
	InputText          string  `json:"input_text"`
	Type               string  `json:"type"`
	Status             int     `json:"status"` // 1=processing, 2=completed, 3=failed
	StatusDesc         string  `json:"status_desc"`
	StatusPercentage   int     `json:"status_percentage"`
	ErrorCode          string  `json:"error_code"`
	ErrorMessage       string  `json:"error_message"`
	ExpiredAt          *string `json:"expired_at"`
	Name               *string `json:"name"`
	EstimatedCredit    int     `json:"estimated_credit"`
	MediaType          string  `json:"media_type"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          *string `json:"updated_at"`
	DelaySeconds       int     `json:"delay_seconds"`
}

type historyListResponse struct {
	Success bool           `json:"success"`
	Total   int            `json:"total"`
	Result  []historyItem `json:"result"`
}

type referenceItem struct {
	MediaType     string `json:"media_type,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
	VideoURL      string `json:"video_url,omitempty"`
	Status        int    `json:"status,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type historyItem struct {
	ID             int              `json:"id"`
	UUID           string           `json:"uuid"`
	UserID         int              `json:"user_id"`
	ModelName      string           `json:"model_name"`
	InputText      string           `json:"input_text"`
	NegativePrompt *string          `json:"negative_prompt"`
	Type           string           `json:"type"`
	Status         int              `json:"status"`
	StatusDesc     string           `json:"status_desc"`
	StatusPercent  int              `json:"status_percentage"`
	ErrorCode      string           `json:"error_code"`
	ErrorMessage   string           `json:"error_message"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at,omitempty"`
	ThumbnailURL   string           `json:"thumbnail_url,omitempty"`
	ReferenceItem  []referenceItem  `json:"reference_item,omitempty"`
	GeneratedVideo []generatedVideo `json:"generated_video,omitempty"`
	GeneratedImage []generatedImage `json:"generated_image,omitempty"`
}

type generatedVideo struct {
	ID          int     `json:"id"`
	UUID        string  `json:"uuid"`
	HistoryID   int     `json:"history_id"`
	VideoURI    string  `json:"video_uri"`
	Duration    float64 `json:"duration"`
	AspectRatio string  `json:"aspect_ratio"`
	Resolution  string  `json:"resolution"`
	Status      int     `json:"status"`
	VideoURL    string  `json:"video_url"`
}

type generatedImage struct {
	ID         int    `json:"id"`
	UUID       string `json:"uuid"`
	ImageURL   string `json:"image_url"`
	ImageURI   string `json:"image_uri"`
	Status     int    `json:"status"`
	Model      string `json:"model"`
	Resolution string `json:"resolution"`
}

const maxVeoImageSize = 20 * 1024 * 1024 // 20 MB per image

func extFromMime(mime string) string {
	switch strings.ToLower(mime) {
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
	default:
		return "bin"
	}
}

func isImageURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") ||
		strings.HasSuffix(lower, ".webp") || strings.HasSuffix(lower, ".bmp") ||
		strings.HasSuffix(lower, ".heic") || strings.HasSuffix(lower, ".heif")
}

type veoParameters struct {
	Prompt       string   `json:"prompt,omitempty"`
	Model        string   `json:"model,omitempty"`
	Resolution   string   `json:"resolution,omitempty"`
	AspectRatio  string   `json:"aspect_ratio,omitempty"`
	ModeImage    string   `json:"mode_image,omitempty"`
	RefImages    []string `json:"ref_images,omitempty"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateMultipartDirect(c, info)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	seconds, _ := strconv.Atoi(req.Seconds)
	if seconds == 0 {
		seconds = req.Duration
	}
	if seconds <= 0 {
		seconds = 8
	}
	size := req.Size
	if size == "" {
		size = "1280x720"
	}
	resRatio := 1.0
	switch size {
	case "1920x1080":
		resRatio = 2.25
	default: // 1280x720
		resRatio = 1.0
	}
	return map[string]float64{
		"seconds":    float64(seconds),
		"resolution": resRatio,
	}
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	model := info.UpstreamModelName
	path, ok := taskModelToPath[model]
	if !ok {
		path = fmt.Sprintf("/uapi/v1/video-gen/%s", model)
	}
	return a.baseURL + path, nil
}

var taskModelToPath = map[string]string{
	// Video generation
	"veo-3.1":       "/uapi/v1/video-gen/veo",
	"veo-3.1-fast": "/uapi/v1/video-gen/veo",
	"veo-2":         "/uapi/v1/video-gen/veo",
	"veo-3.1-lite":  "/uapi/v1/video-gen/veo",
	"grok-3":        "/uapi/v1/video-gen/grok",
	"grok-video":    "/uapi/v1/video-gen/grok",
	"seedance-2":       "/uapi/v1/video-gen/seedance",
	"seedance-2-remix": "/uapi/v1/video-gen/seedance",
	"seedance-2-omni":  "/uapi/v1/video-gen/seedance",
	"kling":            "/uapi/v1/video-gen/kling",
	// Image generation
	"nano-banana-pro": "/uapi/v1/generate_image",
	"nano-banana-2":   "/uapi/v1/generate_image",
	"imagen-4":       "/uapi/v1/generate_image",
	"grok-image":     "/uapi/v1/imagen/grok",
	"meta-ai-image":  "/uapi/v1/meta_ai/generate",
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_body_bytes_failed")
	}
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]interface{}
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			if model, ok := bodyMap["model"].(string); ok && model != "" {
				// model already set, pass through
			} else {
				bodyMap["model"] = info.UpstreamModelName
			}
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		modelName := info.UpstreamModelName
		if v, ok := formData.Value["model"]; ok && len(v) > 0 && v[0] != "" {
			modelName = v[0]
		}

		prompt := ""
		if v, ok := formData.Value["prompt"]; ok && len(v) > 0 {
			prompt = v[0]
		}

		resolution := ""
		if v, ok := formData.Value["resolution"]; ok && len(v) > 0 {
			resolution = v[0]
		}

		aspectRatio := ""
		if v, ok := formData.Value["aspect_ratio"]; ok && len(v) > 0 {
			aspectRatio = v[0]
		}

		modeImage := ""
		if v, ok := formData.Value["mode_image"]; ok && len(v) > 0 {
			modeImage = v[0]
		}

		mode := ""
		if v, ok := formData.Value["mode"]; ok && len(v) > 0 {
			mode = v[0]
		}

		duration := ""
		if v, ok := formData.Value["duration"]; ok && len(v) > 0 {
			duration = v[0]
		}

		writer.WriteField("prompt", prompt)
		writer.WriteField("model", modelName)
		if resolution != "" {
			writer.WriteField("resolution", resolution)
		}
		if aspectRatio != "" {
			writer.WriteField("aspect_ratio", aspectRatio)
		}
		if modeImage != "" {
			writer.WriteField("mode_image", modeImage)
		}
		debugLog := fmt.Sprintf("[SeedanceDebug] mode=%s, duration=%s, ref_images_file=%d, ref_images_value=%d\n", mode, duration, len(formData.File["ref_images"]), len(formData.Value["ref_images"]))
		os.WriteFile("/tmp/seedance_debug.log", []byte(debugLog), 0644)
		if mode != "" {
			writer.WriteField("mode", mode)
		}
		if duration != "" {
			writer.WriteField("duration", duration)
		}

		// Models whose upstream expects reference images in "files" field (not "ref_images").
		needsFilesField := strings.HasPrefix(modelName, "grok-") || strings.HasPrefix(modelName, "nano-banana-") || modelName == "imagen-4"

		// Handle ref_images and files (for reference images)
		for fieldName, fileHeaders := range formData.File {
			if fieldName != "ref_images" && fieldName != "files" && fieldName != "ref_videos" && fieldName != "ref_audios" {
				continue
			}
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					ct = http.DetectContentType(buf512[:n])
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}

				// grok / nano-banana / imagen-4: local images must use "files" field, not "ref_images".
				upstreamField := fieldName
				if needsFilesField && fieldName == "ref_images" {
					upstreamField = "files"
				}

				h := make(textproto.MIMEHeader)
				// Preserve original field name (ref_images/ref_videos/ref_audios/files)
				// so upstream can correctly identify reference media types.
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, upstreamField, fh.Filename))
				h.Set("Content-Type", ct)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				io.Copy(part, f)
				f.Close()
			}
		}

		// Handle ref_images as text values: base64 data URL, plain base64, or HTTP URL.
		// - HTTP URL with image extension -> forward as file_urls text field (upstream fetches it)
		// - base64 or data: URI -> decode and send as "files" multipart part
		for fieldName, values := range formData.Value {
			// file_urls: forward directly to upstream as-is (upstream expects array)
			if fieldName == "file_urls" {
				for _, val := range values {
					writer.WriteField("file_urls", val)
				}
				continue
			}
			if fieldName != "ref_images" && fieldName != "files" {
				continue
			}
			for i, val := range values {
				if val == "" {
					continue
				}

				// grok / nano-banana / imagen-4: text values have different field mappings.
				// - HTTP URLs  -> file_urls
				// - data: URI  -> decode and send as files multipart part
				// - other text (UUID etc) -> keep as ref_images
				if needsFilesField {
					if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
						writer.WriteField("file_urls", val)
						continue
					}
					if strings.HasPrefix(val, "data:") {
						// data: URI -> decode and send as files multipart part
						rest := val[len("data:"):]
						mimeType := "application/octet-stream"
						if idx := strings.Index(rest, ","); idx >= 0 {
							mimeType = rest[:idx]
							if sem := strings.Index(mimeType, ";"); sem >= 0 {
								mimeType = mimeType[:sem]
							}
							b64str := rest[idx+1:]
							data, _ := base64.StdEncoding.DecodeString(b64str)
							if len(data) > 0 && len(data) <= maxVeoImageSize {
								h := make(textproto.MIMEHeader)
								filename := fmt.Sprintf("ref_image_%d_%d.%s", time.Now().UnixNano(), i, extFromMime(mimeType))
								h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, filename))
								h.Set("Content-Type", mimeType)
								part, err := writer.CreatePart(h)
								if err == nil {
									part.Write(data)
								}
							}
						}
						continue
					}
					// UUID or other string -> keep as ref_images
					writer.WriteField("ref_images", val)
					continue
				}

				var data []byte
				mimeType := "application/octet-stream"

				if strings.HasPrefix(val, "data:") {
					// data:image/png;base64,... -> decode and send as files part
					rest := val[len("data:"):]
					if idx := strings.Index(rest, ","); idx >= 0 {
						mimeType = rest[:idx]
						if sem := strings.Index(mimeType, ";"); sem >= 0 {
							mimeType = mimeType[:sem]
						}
						b64str := rest[idx+1:]
						data, _ = base64.StdEncoding.DecodeString(b64str)
					}
				} else if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
					// All HTTP URLs (including images): download and send as files part.
					// Upstream does not accept URL references for ref_images; it requires
					// actual file content in the multipart body.
					resp, err := http.Get(val)
					if err == nil && resp.StatusCode == 200 {
						data, _ = io.ReadAll(resp.Body)
						resp.Body.Close()
						if ct := resp.Header.Get("Content-Type"); ct != "" {
							mimeType = ct
						}
					}
				} else {
					// plain base64 -> decode and send as files part
					data, _ = base64.StdEncoding.DecodeString(val)
				}

				if len(data) == 0 || len(data) > maxVeoImageSize {
					continue
				}
				h := make(textproto.MIMEHeader)
				filename := fmt.Sprintf("ref_image_%d_%d.%s", time.Now().UnixNano(), i, extFromMime(mimeType))
				// Use original field name (ref_images/files) so upstream can identify it.
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
				h.Set("Content-Type", mimeType)
				part, err := writer.CreatePart(h)
				if err != nil {
					continue
				}
				part.Write(data)
			}
		}

		writer.Close()
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return common.ReaderOnly(storage), nil
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

	var dResp submitResponse
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.UUID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("uuid is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.UUID = dResp.UUID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	// Use uuid as upstream task ID for polling
	return dResp.UUID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}

	// Step 1: Find the history uuid from the history list (filter by all to include image tasks)
	listURL := fmt.Sprintf("%s/uapi/v1/histories?filter_by=all&items_per_page=50&page=1", baseUrl)
	req, _ := http.NewRequest(http.MethodGet, listURL, nil)
	req.Header.Set("x-api-key", key)
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return resp, err
	}
	listBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var listResp historyListResponse
	if err := common.Unmarshal(listBody, &listResp); err != nil || !listResp.Success {
		return nil, fmt.Errorf("history list failed")
	}

	// Find the matching history item by UUID first, then numeric id fallback
	var historyUUID string
	for _, item := range listResp.Result {
		if item.UUID == taskID || strconv.Itoa(item.ID) == taskID {
			historyUUID = item.UUID
			break
		}
	}

	if historyUUID == "" {
		return nil, fmt.Errorf("history not found for task_id: %s", taskID)
	}

	// Step 2: Get detailed history with video URL
	detailURL := fmt.Sprintf("%s/uapi/v1/history/%s", baseUrl, historyUUID)
	req2, _ := http.NewRequest(http.MethodGet, detailURL, nil)
	req2.Header.Set("x-api-key", key)
	return client.Do(req2)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"veo-3.1", "veo-3.1-fast", "veo-2", "veo-3.1-lite", "grok-3", "grok-video", "seedance-2", "seedance-2-remix", "seedance-2-omni", "kling"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "GeminiGen"
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// The history detail response shares the same fields as historyItem
	var h historyItem
	if err := common.Unmarshal(respBody, &h); err != nil {
		return nil, errors.Wrap(err, "unmarshal history detail failed")
	}

	taskResult := &relaycommon.TaskInfo{
		Code:   0,
		TaskID: strconv.Itoa(h.ID),
	}

	switch h.Status {
	case 1: // processing
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = fmt.Sprintf("%d%%", h.StatusPercent)
		// Some platforms (e.g. grok) may return video/image URLs even while status is still processing.
		if len(h.GeneratedVideo) > 0 && h.GeneratedVideo[0].VideoURL != "" {
			taskResult.Url = h.GeneratedVideo[0].VideoURL
		}
		if len(h.GeneratedImage) > 0 && h.GeneratedImage[0].ImageURL != "" {
			taskResult.Url = h.GeneratedImage[0].ImageURL
		}
		if len(h.ReferenceItem) > 0 {
			refs := make([]interface{}, len(h.ReferenceItem))
			for i, r := range h.ReferenceItem {
				refs[i] = r
			}
			taskResult.ReferenceItem = refs
		}
	case 2: // completed
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		// Extract video URL from generated_video array
		if len(h.GeneratedVideo) > 0 && h.GeneratedVideo[0].VideoURL != "" {
			taskResult.Url = h.GeneratedVideo[0].VideoURL
		}
		// Extract image URL from generated_image array
		if len(h.GeneratedImage) > 0 && h.GeneratedImage[0].ImageURL != "" {
			taskResult.Url = h.GeneratedImage[0].ImageURL
		}
		// Pass reference_item for downstream consumers
		if len(h.ReferenceItem) > 0 {
			refs := make([]interface{}, len(h.ReferenceItem))
			for i, r := range h.ReferenceItem {
				refs[i] = r
			}
			taskResult.ReferenceItem = refs
		}
	case 3: // failed
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if h.ErrorMessage != "" {
			taskResult.Reason = h.ErrorMessage
		} else {
			taskResult.Reason = h.ErrorCode
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
	}

	return taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var dResp submitResponse
	if err := common.Unmarshal(task.Data, &dResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task data failed")
	}

	// Use base method which reads ResultURL from task.PrivateData.ResultURL
	// (set by relay_task.go after ParseTaskResult)
	video := task.ToOpenAIVideo()
	// Override model-specific fields from stored submit response
	if dResp.ModelName != "" {
		video.Model = dResp.ModelName
	}
	video.Status = task.Status.ToVideoStatus()
	video.SetProgressStr(task.Progress)
	if task.FinishTime > 0 {
		video.CompletedAt = task.FinishTime
	} else if task.UpdatedAt > 0 {
		video.CompletedAt = task.UpdatedAt
	}
	// ResultURL (video URL) is already set by ToOpenAIVideo via GetResultURL()
	return common.Marshal(video)
}
