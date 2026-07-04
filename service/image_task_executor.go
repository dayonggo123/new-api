package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// ImageTaskAdaptor is the minimal adaptor interface needed by the image task
// executor. It mirrors the relevant methods of relay/channel.Adaptor and is used
// to break the service -> relay -> relay/channel -> service import cycle.
type ImageTaskAdaptor interface {
	Init(info *relaycommon.RelayInfo)
	ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error)
	DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error)
	DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError)
}

// GetImageTaskAdaptorFunc is injected by main.go to resolve an adaptor by its
// API type without creating a package cycle.
var GetImageTaskAdaptorFunc func(apiType int) ImageTaskAdaptor

// ImageTaskExecutor executes a single image generation task against the
// upstream channel.
type ImageTaskExecutor struct {
	queue ImageTaskQueue
	retry ImageTaskRetryPolicy
}

// NewImageTaskExecutor creates a new executor with the default retry policy.
func NewImageTaskExecutor(queue ImageTaskQueue) *ImageTaskExecutor {
	return &ImageTaskExecutor{
		queue: queue,
		retry: NewDefaultImageTaskRetryPolicy(),
	}
}

// Execute replays the image request from the task, calls the upstream channel,
// rewrites image URLs with local proxy URLs, and updates the task status.
func (e *ImageTaskExecutor) Execute(ctx context.Context, task *model.Task) error {
	requestPayload := []byte(task.PrivateData.RequestPayload)

	c, recorder, err := BuildFakeGinContext(ctx, task, requestPayload)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("build fake gin context: %w", err))
	}

	info, err := BuildRelayInfoFromTask(c, task)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("build relay info from task: %w", err))
	}

	adaptor := GetImageTaskAdaptorFunc(info.ApiType)
	if adaptor == nil {
		return e.handleFailure(ctx, task, fmt.Errorf("adaptor not found for api type %d", info.ApiType))
	}
	adaptor.Init(info)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok || imageReq == nil {
		return e.handleFailure(ctx, task, fmt.Errorf("invalid request type in task %s", task.TaskID))
	}

	requestCopy, err := common.DeepCopy(imageReq)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("copy image request: %w", err))
	}

	convertedRequest, err := adaptor.ConvertImageRequest(c, info, *requestCopy)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("convert image request: %w", err))
	}

	requestBody, err := e.buildRequestBody(info, convertedRequest)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("build request body: %w", err))
	}

	rawResp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return e.handleFailure(ctx, task, fmt.Errorf("upstream request failed: %w", err))
	}

	httpResp, ok := rawResp.(*http.Response)
	if !ok {
		return e.handleFailure(ctx, task, fmt.Errorf("upstream response is not *http.Response"))
	}

	if httpResp.StatusCode != http.StatusOK {
		apiErr := types.NewOpenAIError(
			fmt.Errorf("upstream returned status %d", httpResp.StatusCode),
			types.ErrorCodeBadResponseStatusCode,
			httpResp.StatusCode,
		)
		return e.handleFailure(ctx, task, apiErr)
	}

	// Rewrite upstream image URLs to persistent local proxy URLs. The fake gin
	// context carries the downstream base URL so that the generated URLs are
	// valid for the client.
	httpResp = RewriteImageResponseWithProxyURLs(c, httpResp)

	usage, apiErr := adaptor.DoResponse(c, httpResp, info)
	if apiErr != nil {
		return e.handleFailure(ctx, task, apiErr)
	}
	_ = usage

	capturedBody := CaptureImageResponse(recorder)
	if len(capturedBody) == 0 {
		return e.handleFailure(ctx, task, fmt.Errorf("empty image response body"))
	}

	resultURL := ExtractImageURLFromResponse(capturedBody)

	ok, err = e.queue.MarkSuccess(task, capturedBody, resultURL)
	if err != nil {
		return fmt.Errorf("mark task success failed: %w", err)
	}
	if !ok {
		// Another worker won the CAS; the task has already been moved out of
		// IN_PROGRESS, so we can safely discard the result.
		common.SysLog(fmt.Sprintf("[ImageTaskExecutor] task %s already moved out of IN_PROGRESS, discarding success", task.TaskID))
		return nil
	}

	// Settle the pre-consumed billing. Since the pre-consumed quota was already
	// calculated from the request, the actual quota is usually equal. In future
	// iterations we can compute a more accurate quota from the upstream usage.
	actualQuota := e.resolveActualQuota(info, task, usage)
	if settleErr := SettleBilling(ctx, info, actualQuota); settleErr != nil {
		common.SysError(fmt.Sprintf("[ImageTaskExecutor] settle billing for task %s failed: %v", task.TaskID, settleErr))
	}

	return nil
}

// buildRequestBody serializes the converted request into an io.Reader. For
// multipart requests it expects the adaptor to have returned the body as an
// io.Reader.
func (e *ImageTaskExecutor) buildRequestBody(info *relaycommon.RelayInfo, convertedRequest any) (io.Reader, error) {
	switch v := convertedRequest.(type) {
	case *bytes.Buffer:
		return v, nil
	case io.Reader:
		return v, nil
	default:
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return nil, fmt.Errorf("marshal converted request: %w", err)
		}
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return nil, fmt.Errorf("apply param override: %w", err)
			}
		}
		return bytes.NewBuffer(jsonData), nil
	}
}

// handleFailure decides whether a failed task should be retried or marked as
// permanently failed. It always returns the original error so that the worker
// can log it.
func (e *ImageTaskExecutor) handleFailure(ctx context.Context, task *model.Task, err error) error {
	if e.retry.ShouldRetry(task, err) {
		nextRetryAt := e.retry.NextRetryAt(task.PrivateData.RetryCount)
		reason := err.Error()
		ok, markErr := e.queue.MarkRetry(task, reason, nextRetryAt)
		if markErr != nil {
			common.SysError(fmt.Sprintf("[ImageTaskExecutor] MarkRetry for task %s failed: %v", task.TaskID, markErr))
		} else if ok {
			common.SysLog(fmt.Sprintf("[ImageTaskExecutor] task %s scheduled for retry %d at %d: %s", task.TaskID, task.PrivateData.RetryCount, nextRetryAt, reason))
		}
		return err
	}

	ok, markErr := e.queue.MarkFailure(task, err.Error())
	if markErr != nil {
		common.SysError(fmt.Sprintf("[ImageTaskExecutor] MarkFailure for task %s failed: %v", task.TaskID, markErr))
		return err
	}
	if !ok {
		common.SysLog(fmt.Sprintf("[ImageTaskExecutor] task %s already moved out of IN_PROGRESS, discarding failure", task.TaskID))
		return err
	}

	RefundTaskQuota(ctx, task, err.Error())
	return err
}

// resolveActualQuota returns the quota to settle against the pre-consumed
// amount. For now the pre-consumed value is used as the actual quota because
// image generation costs are determined by the request parameters (model, size,
// quality, n) that are known at submission time.
func (e *ImageTaskExecutor) resolveActualQuota(info *relaycommon.RelayInfo, task *model.Task, usage any) int {
	if usage == nil {
		return task.Quota
	}
	u, ok := usage.(*dto.Usage)
	if !ok || u == nil {
		return task.Quota
	}
	// Only adjust when the upstream returned meaningful usage. Most image
	// channels do not return usage, so we keep the pre-consumed amount.
	if u.TotalTokens == 0 {
		return task.Quota
	}
	// TODO: compute actual quota from usage and info.PriceData when channels
	// start returning real image usage.
	return task.Quota
}
