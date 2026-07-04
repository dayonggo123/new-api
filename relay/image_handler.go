package relay

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
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func isTaskImageChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeAPIMart, constant.ChannelTypeDuoYuanTanSuo, constant.ChannelTypeZhangyuge,
		constant.ChannelTypeVeo: // GeminiGen: nano-banana / imagen 等图像模型也走异步 task 路径
		return true
	}
	return false
}

func handleTaskImageRelay(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	result, taskErr := RelayTaskSubmit(c, info)
	if taskErr != nil {
		return types.NewErrorWithStatusCode(taskErr.Error, types.ErrorCodeBadResponseStatusCode, taskErr.StatusCode, types.ErrOptionWithSkipRetry())
	}

	// 插入任务（必须在结算和返回客户端前成功落库）
	task := model.InitTask(result.Platform, info)
	task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
	task.PrivateData.BillingSource = info.BillingSource
	task.PrivateData.SubscriptionId = info.SubscriptionId
	task.PrivateData.TokenId = info.TokenId
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		ModelPrice:      info.PriceData.ModelPrice,
		GroupRatio:      info.PriceData.GroupRatioInfo.GroupRatio,
		ModelRatio:      info.PriceData.ModelRatio,
		OtherRatios:     info.PriceData.OtherRatios,
		OriginModelName: info.OriginModelName,
		PerCallBilling:  common.StringsContains(constant.TaskPricePatches, info.OriginModelName) || info.PriceData.UsePrice,
	}
	task.Quota = result.Quota
	task.Data = result.TaskData
	task.Action = info.Action
	if insertErr := task.Insert(); insertErr != nil {
		common.SysError("insert task error: " + insertErr.Error())
		// 任务落库失败，上游任务可能已提交；退款并返回错误，避免客户端看到虚假成功
		if info.Billing != nil {
			info.Billing.Refund(c)
		}
		return types.NewErrorWithStatusCode(insertErr, types.ErrorCodeBadResponseStatusCode, http.StatusInternalServerError, types.ErrOptionWithSkipRetry())
	}

	// 结算
	if settleErr := service.SettleBilling(c, info, result.Quota); settleErr != nil {
		common.SysError("settle task billing error: " + settleErr.Error())
	}
	service.LogTaskConsumption(c, info)

	// Register to async_image system so downstream clients polling /v1/images/tasks/{task_id} can find it
	service.RegisterAsyncImageTask(info.PublicTaskID, info)
	if result.UpstreamTaskID != "" {
		service.SetAsyncImageTaskUpstreamID(info.PublicTaskID, result.UpstreamTaskID)
	}

	return nil
}

func isSyncImageAsyncChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeGemini, constant.ChannelTypeVolcEngine:
		return true
	}
	return false
}

// syncImageResponseWriter captures the response body of a synchronous image generation
// without forwarding it to the downstream client. This lets us store the result as a
// completed task and return a task_id instead.
type syncImageResponseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	header     http.Header
	written    bool
}

func newSyncImageResponseWriter(w gin.ResponseWriter) *syncImageResponseWriter {
	return &syncImageResponseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
		header:         http.Header{},
	}
}

func (w *syncImageResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
}

func (w *syncImageResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *syncImageResponseWriter) Header() http.Header {
	return w.header
}

func (w *syncImageResponseWriter) Status() int {
	return w.statusCode
}

func ImageHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	// 去掉 provider 前缀（如 newapi/），避免影响模型名匹配和上游请求
	if idx := strings.Index(info.OriginModelName, "/"); idx >= 0 {
		info.OriginModelName = info.OriginModelName[idx+1:]
	}

	// 对于 APIMart/DuoYuanTanSuo/GeminiGen 的 task 模型，走 task 异步流程
	// gpt-image (APIMart) 和 nano-banana (GeminiGen/Veo) 都走 task 路径
	if isTaskImageChannel(info.ChannelType) &&
		(strings.HasPrefix(info.OriginModelName, "gpt-image") ||
			strings.HasPrefix(info.OriginModelName, "nano-banana-")) {
		return handleTaskImageRelay(c, info)
	}

	// 火山方舟 / OpenAI / Google Gemini 的图片生成也包装成异步任务，
	// 使下游可以通过 /v1/images/tasks/{task_id} 查询结果。
	if isSyncImageAsyncChannel(info.ChannelType) &&
		(info.RelayMode == relayconstant.RelayModeImagesGenerations ||
			info.RelayMode == relayconstant.RelayModeImagesEdits) {
		return handleSyncImageAsTaskRelay(c, info)
	}

	return runSyncImageRelay(c, info)
}

// runSyncImageRelay executes the original synchronous image generation flow.
// It is extracted from ImageHelper so that handleSyncImageAsTaskRelay can capture its output.
func runSyncImageRelay(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	var requestBody io.Reader

	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		requestBody = common.ReaderOnly(storage)
	} else {
		convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed)
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		switch convertedRequest.(type) {
		case *bytes.Buffer:
			requestBody = convertedRequest.(io.Reader)
		default:
			jsonData, err := common.Marshal(convertedRequest)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			// apply param override
			if len(info.ParamOverride) > 0 {
				jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
				if err != nil {
					return newAPIErrorFromParamOverride(err)
				}
			}

			if common.DebugEnabled {
				logger.LogDebug(c, fmt.Sprintf("image request body: %s", string(jsonData)))
			}
			requestBody = bytes.NewBuffer(jsonData)
		}
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
				// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
				httpResp.StatusCode = http.StatusOK
			} else {
				newAPIError := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
				// reset status code 重置状态码
				service.ResetStatusCode(newAPIError, statusCodeMappingStr)
				return newAPIError
			}
		}
	}

	// Async mode: if upstream returns a task_id, return it directly
	// and let the client poll /v1/images/tasks/{task_id} for the result.
	if c.Query("async") == "true" && httpResp != nil {
		body, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if readErr != nil {
			return types.NewOpenAIError(readErr, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
		}
		var asyncResp map[string]any
		if err := common.Unmarshal(body, &asyncResp); err == nil {
			taskID := ""
			if t, ok := asyncResp["task_id"].(string); ok && t != "" {
				taskID = t
			} else if t, ok := asyncResp["id"].(string); ok && t != "" {
				taskID = t
			}
			if taskID != "" {
				service.RegisterAsyncImageTask(taskID, info)
				c.JSON(http.StatusOK, asyncResp)
				return nil
			}
		}
		// Not an async response, restore body for normal processing
		httpResp.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Intercept image generation responses to replace temporary upstream URLs
	// with persistent local proxy URLs.
	httpResp = rewriteImageResponseWithProxyURLs(c, httpResp)

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	imageN := uint(1)
	if request.N != nil {
		imageN = *request.N
	}

	// n is handled via OtherRatio so it is applied exactly once in quota
	// calculation (both price-based and ratio-based paths).
	// Adaptors may have already set a more accurate count from the
	// upstream response; only set the default when they haven't.
	if _, hasN := info.PriceData.OtherRatios["n"]; !hasN {
		info.PriceData.AddOtherRatio("n", float64(imageN))
	}

	if usage.(*dto.Usage).TotalTokens == 0 {
		usage.(*dto.Usage).TotalTokens = 1
	}
	if usage.(*dto.Usage).PromptTokens == 0 {
		usage.(*dto.Usage).PromptTokens = 1
	}

	quality := "standard"
	if request.Quality == "hd" {
		quality = "hd"
	}

	var logContent []string

	if len(request.Size) > 0 {
		logContent = append(logContent, fmt.Sprintf("大小 %s", request.Size))
	}
	if len(quality) > 0 {
		logContent = append(logContent, fmt.Sprintf("品质 %s", quality))
	}
	if imageN > 0 {
		logContent = append(logContent, fmt.Sprintf("生成数量 %d", imageN))
	}

	service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), logContent)
	return nil
}

// handleSyncImageAsTaskRelay runs a synchronous image generation but stores the result
// as a completed task and returns a task_id to the downstream client.
func handleSyncImageAsTaskRelay(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	// Generate public task ID
	publicTaskID := model.GenerateTaskID()
	info.PublicTaskID = publicTaskID
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	info.TaskRelayInfo.PublicTaskID = publicTaskID

	// Capture the synchronous image response so we can store it as task data
	// and return a task_id to the client instead of the raw image.
	recorder := newSyncImageResponseWriter(c.Writer)
	c.Writer = recorder

	// Run the existing synchronous image generation path
	err := runSyncImageRelay(c, info)

	// Restore original writer
	c.Writer = recorder.ResponseWriter

	if err != nil {
		return err
	}

	if recorder.statusCode != http.StatusOK {
		// Upstream returned a non-200 success code. Pass it through as-is.
		recorder.ResponseWriter.WriteHeader(recorder.statusCode)
		if recorder.body.Len() > 0 {
			recorder.ResponseWriter.Write(recorder.body.Bytes())
		}
		return nil
	}

	capturedBody := recorder.body.Bytes()
	if len(capturedBody) == 0 {
		return types.NewError(fmt.Errorf("empty image response"), types.ErrorCodeBadResponseBody, types.ErrOptionWithSkipRetry())
	}

	// Create task record with the captured image response as completed data
	platform := constant.TaskPlatform(strconv.Itoa(info.ChannelType))
	task := model.InitTask(platform, info)
	task.Status = model.TaskStatusSuccess
	task.Progress = "100%"
	task.StartTime = task.SubmitTime
	task.FinishTime = time.Now().Unix()
	task.Action = constant.TaskActionGenerate
	task.Data = capturedBody
	task.Quota = info.PriceData.Quota
	if task.Quota == 0 {
		task.Quota = info.PriceData.QuotaToPreConsume
	}

	// Extract result URL from the captured image response for convenience
	if url := extractImageURLFromResponse(capturedBody); url != "" {
		task.PrivateData.ResultURL = url
	}

	if insertErr := task.Insert(); insertErr != nil {
		common.SysError("insert sync image task error: " + insertErr.Error())
		return types.NewErrorWithStatusCode(insertErr, types.ErrorCodeBadResponseStatusCode, http.StatusInternalServerError, types.ErrOptionWithSkipRetry())
	}

	// Register in async image system so polling can find it
	service.RegisterAsyncImageTask(publicTaskID, info)

	// Return OpenAI-compatible task response to the client
	resp := dto.NewOpenAIVideo()
	resp.ID = publicTaskID
	resp.TaskID = publicTaskID
	resp.Status = dto.VideoStatusCompleted
	resp.Progress = 100
	resp.CreatedAt = task.SubmitTime
	resp.CompletedAt = task.FinishTime
	resp.Model = info.OriginModelName
	if task.PrivateData.ResultURL != "" {
		resp.SetMetadata("url", task.PrivateData.ResultURL)
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

// extractImageURLFromResponse extracts the first image URL from an OpenAI-compatible
// image response, if present.
func extractImageURLFromResponse(body []byte) string {
	var imgResp dto.ImageResponse
	if err := common.Unmarshal(body, &imgResp); err != nil {
		return ""
	}
	for _, item := range imgResp.Data {
		if item.Url != "" {
			return item.Url
		}
	}
	return ""
}

// rewriteImageResponseWithProxyURLs reads the upstream image generation response,
// replaces temporary upstream image URLs with persistent local proxy URLs,
// and returns a new http.Response with the modified body.
func rewriteImageResponseWithProxyURLs(c *gin.Context, resp *http.Response) *http.Response {
	if resp == nil || resp.Body == nil {
		return resp
	}
	// Only rewrite JSON image responses (skip b64_json, stream, etc.)
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return resp
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return resp
	}

	var imgResp dto.ImageResponse
	if err := common.Unmarshal(body, &imgResp); err != nil {
		// Not a valid image response, restore body and return as-is
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	modified := false
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	baseURL := scheme + "://" + host

	for i := range imgResp.Data {
		if imgResp.Data[i].Url != "" && imgResp.Data[i].B64Json == "" {
			proxyID := service.RegisterImageProxyURL(imgResp.Data[i].Url)
			imgResp.Data[i].Url = baseURL + "/image-proxy/" + proxyID + ".png"
			modified = true
		}
	}

	if !modified {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	newBody, err := common.Marshal(imgResp)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}
	resp.Body = io.NopCloser(bytes.NewReader(newBody))
	resp.ContentLength = int64(len(newBody))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))
	return resp
}
