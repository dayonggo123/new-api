package wanxiang

import (
	"bytes"
	"encoding/base64"
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
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type submitRequest struct {
	Model  string                 `json:"model"`
	Prompt string                 `json:"prompt,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type submitResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskID interface{} `json:"task_id"`
	} `json:"data"`
}

type statusResponse struct {
	TaskID         interface{} `json:"task_id"`
	State          string      `json:"state"`
	Status         string      `json:"status"`
	IsFinal        bool        `json:"is_final"`
	Progress       string      `json:"progress"`
	ResultURL      string      `json:"result_url"`
	ResultType     string      `json:"result_type"`
	Cost           float64     `json:"cost"`
	Error          interface{} `json:"error"`
	Refunded       bool        `json:"refunded"`
	RefundedAmount float64     `json:"refunded_amount"`
	CreatedAt      string      `json:"created_at"`
	CompletedAt    string      `json:"completed_at"`
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
	// Allow action override via metadata for flexibility
	if metaAction, ok := req.Metadata["action"]; ok {
		if actionStr, ok := metaAction.(string); ok && actionStr != "" {
			info.Action = actionStr
		}
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	common.SysLog(fmt.Sprintf("[WanXiang] BuildRequestBody: images=%d image=%q content-type=%q",
		len(req.Images), req.Image, c.GetHeader("Content-Type")))

	// If no images in task_request but original request is multipart,
	// extract image data from raw body and convert to base64 data URLs.
	// WanXiangAI upstream does not support downloading external HTTP URLs,
	// so we must embed image content directly as data URLs in the JSON body.
	if len(req.Images) == 0 && req.Image == "" && strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
		if dataURLs, err := extractMultipartImagesAsDataURLs(c); err == nil && len(dataURLs) > 0 {
			req.Images = dataURLs
			common.SysLog(fmt.Sprintf("[WanXiang] extracted %d images from multipart as data URLs", len(dataURLs)))
		} else if err != nil {
			common.SysLog(fmt.Sprintf("[WanXiang] extractMultipartImagesAsDataURLs failed: %v", err))
		} else {
			common.SysLog("[WanXiang] no image files found in multipart")
		}
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, err
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/media/generate", a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
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

	var sResp submitResponse
	err = common.Unmarshal(responseBody, &sResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrap(err, fmt.Sprintf("%s", responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if sResp.Code != 200 {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("submit failed: %s", sResp.Msg), "submit_failed", http.StatusBadRequest)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	// task_id from upstream is a number; json unmarshal to interface{} gives float64.
	// Must convert carefully to avoid scientific notation like "2.1949885e+07".
	var upstreamTaskID string
	switch v := sResp.Data.TaskID.(type) {
	case string:
		upstreamTaskID = v
	case float64:
		upstreamTaskID = strconv.FormatInt(int64(v), 10)
	case int:
		upstreamTaskID = strconv.Itoa(v)
	case int64:
		upstreamTaskID = strconv.FormatInt(v, 10)
	default:
		upstreamTaskID = fmt.Sprintf("%v", v)
	}
	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s/v1/skills/task-status?task_id=%s", baseUrl, taskID)

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
	taskInfo := &relaycommon.TaskInfo{}

	var sResp statusResponse
	err := common.Unmarshal(respBody, &sResp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	// Map progress string
	taskInfo.Progress = sResp.Progress

	switch sResp.State {
	case "success":
		taskInfo.Status = model.TaskStatusSuccess
		if sResp.ResultURL != "" {
			taskInfo.Url = sResp.ResultURL
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
		if sResp.Error != nil {
			if errStr, ok := sResp.Error.(string); ok && errStr != "" {
				taskInfo.Reason = errStr
			} else {
				taskInfo.Reason = sResp.Status
			}
		} else if sResp.Status != "" {
			taskInfo.Reason = sResp.Status
		}
	default:
		if sResp.IsFinal {
			// Treat unknown final state as success if result_url exists
			if sResp.ResultURL != "" {
				taskInfo.Status = model.TaskStatusSuccess
				taskInfo.Url = sResp.ResultURL
			} else {
				taskInfo.Status = model.TaskStatusFailure
				taskInfo.Reason = sResp.Status
			}
		} else {
			taskInfo.Status = model.TaskStatusInProgress
		}
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var sResp statusResponse
	if err := common.Unmarshal(originTask.Data, &sResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal wanxiang task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt

	if sResp.ResultURL != "" {
		openAIVideo.SetMetadata("url", sResp.ResultURL)
	}
	if sResp.ResultType != "" {
		openAIVideo.SetMetadata("type", sResp.ResultType)
	}

	if sResp.State == "failed" {
		msg := ""
		if sResp.Error != nil {
			if errStr, ok := sResp.Error.(string); ok {
				msg = errStr
			}
		}
		if msg == "" && sResp.Status != "" {
			msg = sResp.Status
		}
		if msg != "" {
			openAIVideo.Error = &dto.OpenAIVideoError{
				Message: msg,
				Code:    "task_failed",
			}
		}
	}

	return common.Marshal(openAIVideo)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"gemini-3-pro-image-preview",
		"gemini-3.1-flash-image-preview",
		"veo3.1",
		"veo3.1-lite",
		"veo3.1-4k",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "wanxiangai"
}

// ============================
// helpers
// ============================

// extractMultipartImagesAsBase64 parses the original multipart body and converts
// uploaded image files to base64 data URLs. This is needed when switching channels
// because task_request only stores text fields, and file data is lost.
func extractMultipartImagesAsBase64(c *gin.Context) ([]string, error) {
	formData, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	defer formData.RemoveAll()

	var dataURLs []string
	// Check common image field names used by downstream clients
	for _, fieldName := range []string{"ref_images", "images", "files"} {
		if fileHeaders, ok := formData.File[fieldName]; ok {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					continue
				}
				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					ct = http.DetectContentType(data)
				}
				b64 := base64.StdEncoding.EncodeToString(data)
				dataURLs = append(dataURLs, fmt.Sprintf("data:%s;base64,%s", ct, b64))
			}
		}
	}
	return dataURLs, nil
}

// extractMultipartImagesAsDataURLs parses the original multipart body and returns
// base64 data URLs for all image references.
// Handles TWO downstream patterns:
//   1. Text values (data URL / HTTP URL) in formData.Value["ref_images"] / ["images"]
//   2. Binary file parts in formData.File["ref_images"] / ["images"] / ["files"]
//      → read bytes and encode as data:{mime};base64,{data} URL.
func extractMultipartImagesAsDataURLs(c *gin.Context) ([]string, error) {
	formData, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	defer formData.RemoveAll()

	var dataURLs []string

	// ── Case 1: text values (already data URL or HTTP URL) ──
	for _, fieldName := range []string{"ref_images", "images"} {
		if values, ok := formData.Value[fieldName]; ok {
			for _, val := range values {
				if val != "" {
					preview := val
					if len(preview) > 60 {
						preview = preview[:60] + "..."
					}
					dataURLs = append(dataURLs, val)
					common.SysLog(fmt.Sprintf("[WanXiang] extracted text image from %s: %s", fieldName, preview))
				}
			}
		}
	}

	// ── Case 2: binary file parts → base64 data URL ──
	for _, fieldName := range []string{"ref_images", "images", "files"} {
		if fileHeaders, ok := formData.File[fieldName]; ok {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					common.SysLog(fmt.Sprintf("[WanXiang] open file failed: %v", err))
					continue
				}
				data, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					common.SysLog(fmt.Sprintf("[WanXiang] read file failed: %v", err))
					continue
				}

				ct := fh.Header.Get("Content-Type")
				if ct == "" || ct == "application/octet-stream" {
					ct = http.DetectContentType(data)
				}

				b64 := base64.StdEncoding.EncodeToString(data)
				dataURL := fmt.Sprintf("data:%s;base64,%s", ct, b64)
				dataURLs = append(dataURLs, dataURL)
				common.SysLog(fmt.Sprintf("[WanXiang] encoded file part from %s as data URL (%d bytes -> %d chars)", fieldName, len(data), len(dataURL)))
			}
		}
	}
	return dataURLs, nil
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*submitRequest, error) {
	params := make(map[string]interface{})

	// Images
	images := req.Images
	if len(images) == 0 && req.Image != "" {
		images = []string{req.Image}
	}
	if len(images) > 0 {
		params["images"] = images
	}

	// Size → aspectRatio + imageSize mapping
	// Gemini imageSize values: 0.5K, 1K, 2K, 4K
	if req.Size != "" {
		aspectRatio := mapSizeToAspectRatio(req.Size)
		if aspectRatio != "" {
			params["aspectRatio"] = aspectRatio
		}
		imageSize := mapSizeToImageSize(req.Size)
		if imageSize != "" {
			params["imageSize"] = imageSize
		}
	}
	// imageSize is required for Gemini image models, default to 1K
	if _, ok := params["imageSize"]; !ok {
		params["imageSize"] = "1K"
	}

	// Duration
	if req.Duration > 0 {
		params["duration"] = req.Duration
	}

	// Metadata overrides and extra params
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			// Skip internal fields
			if k == "action" || k == "model" {
				continue
			}
			params[k] = v
		}
	}

	// Video models require quality param; image models accept it too.
	// Default to "sd" if not specified. User can override via metadata.
	if _, ok := params["quality"]; !ok {
		params["quality"] = "sd"
	}

	// Veo 3.1 (non-lite) requires generation_mode: "fast" or "pro"
	if strings.Contains(info.UpstreamModelName, "veo3.1") && !strings.Contains(info.UpstreamModelName, "lite") {
		if _, ok := params["generation_mode"]; !ok {
			params["generation_mode"] = "fast"
		}
	}

	return &submitRequest{
		Model:  info.UpstreamModelName,
		Prompt: req.Prompt,
		Params: params,
	}, nil
}

func mapSizeToAspectRatio(size string) string {
	switch size {
	case "256x256", "512x512", "1024x1024":
		return "1:1"
	case "1536x1024":
		return "3:2"
	case "1024x1536":
		return "2:3"
	case "1024x1792":
		return "9:16"
	case "1792x1024":
		return "16:9"
	case "1920x1080":
		return "16:9"
	case "1080x1920":
		return "9:16"
	default:
		return ""
	}
}

func mapSizeToImageSize(size string) string {
	switch size {
	case "256x256", "512x512":
		return "0.5K"
	case "1024x1024", "1024x1536", "1536x1024":
		return "1K"
	case "1024x1792", "1792x1024", "1920x1080", "1080x1920":
		return "2K"
	default:
		return ""
	}
}
