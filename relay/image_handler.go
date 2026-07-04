package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	httpResp = service.RewriteImageResponseWithProxyURLs(c, httpResp)

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

// handleSyncImageAsTaskRelay converts a synchronous image generation request
// (OpenAI / Gemini / VolcEngine) into an asynchronous task. It pre-consumes
// quota, persists a QUEUED task record, returns 200 to the client immediately,
// and lets the worker pool execute the upstream call later.
func handleSyncImageAsTaskRelay(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	publicTaskID := model.GenerateTaskID()
	info.TaskRelayInfo.PublicTaskID = publicTaskID

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(
			fmt.Errorf("failed to copy request to ImageRequest: %w", err),
			types.ErrorCodeInvalidRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	queue := service.NewImageTaskQueue()
	requestPayload := service.SerializeImageRequest(info, request)
	quota := info.PriceData.Quota
	if quota == 0 {
		quota = info.PriceData.QuotaToPreConsume
	}

	task, err := queue.CreateTask(info, requestPayload, quota)
	if err != nil {
		common.SysError("create image task error: " + err.Error())
		if info.Billing != nil {
			info.Billing.Refund(c)
		}
		return types.NewErrorWithStatusCode(
			err,
			types.ErrorCodeBadResponseStatusCode,
			http.StatusInternalServerError,
			types.ErrOptionWithSkipRetry(),
		)
	}

	// Register in the async image system so in-memory polling can find it
	// without a DB lookup.
	service.RegisterAsyncImageTask(publicTaskID, info)

	// Return 200 OK with a queued task reference.
	resp := dto.NewOpenAIVideo()
	resp.ID = publicTaskID
	resp.TaskID = publicTaskID
	resp.Object = "image.generation"
	resp.Status = dto.VideoStatusQueued
	resp.Progress = 0
	resp.CreatedAt = task.SubmitTime
	resp.Model = info.OriginModelName
	c.JSON(http.StatusOK, resp)
	return nil
}
