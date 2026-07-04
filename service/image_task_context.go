package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// SerializeImageRequest serializes an ImageRequest into JSON bytes suitable for
// persisting in a task record. The returned payload is replayed by the worker.
func SerializeImageRequest(info *relaycommon.RelayInfo, req *dto.ImageRequest) []byte {
	if req == nil {
		if info != nil && info.Request != nil {
			if imageReq, ok := info.Request.(*dto.ImageRequest); ok {
				req = imageReq
			}
		}
	}
	if req == nil {
		return []byte("{}")
	}
	b, err := common.Marshal(req)
	if err != nil {
		// Fallback: should never happen for a valid request.
		return []byte("{}")
	}
	return b
}

// BuildRelayInfoFromTask reconstructs a RelayInfo from the persisted task record
// so that the worker can replay the upstream request without the original HTTP
// context. The provided gin.Context is used only to create a minimal execution
// context for the adaptor pipeline.
func BuildRelayInfoFromTask(c *gin.Context, task *model.Task) (*relaycommon.RelayInfo, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}

	channel, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return nil, fmt.Errorf("get channel for task %s failed: %w", task.TaskID, err)
	}

	token, err := model.GetTokenById(task.PrivateData.TokenId)
	if err != nil {
		return nil, fmt.Errorf("get token for task %s failed: %w", task.TaskID, err)
	}

	var imageReq dto.ImageRequest
	if task.PrivateData.RequestPayload != "" {
		if err := common.Unmarshal([]byte(task.PrivateData.RequestPayload), &imageReq); err != nil {
			return nil, fmt.Errorf("unmarshal image request for task %s failed: %w", task.TaskID, err)
		}
	}

	apiType, _ := common.ChannelType2APIType(channel.Type)
	baseURL := ""
	if channel.BaseURL != nil {
		baseURL = *channel.BaseURL
	}

	info := &relaycommon.RelayInfo{
		UserId:              task.UserId,
		UsingGroup:          task.Group,
		UserGroup:           task.Group,
		TokenId:             task.PrivateData.TokenId,
		TokenKey:            token.Key,
		TokenGroup:          task.Group,
		RequestId:           task.TaskID,
		StartTime:           time.Unix(task.SubmitTime, 0),
		FirstResponseTime:   time.Unix(task.SubmitTime, 0).Add(-time.Second),
		OriginModelName:     task.Properties.OriginModelName,
		RelayMode:           task.PrivateData.RelayMode,
		Request:             &imageReq,
		FinalPreConsumedQuota: task.Quota,
		BillingSource:       task.PrivateData.BillingSource,
		SubscriptionId:      task.PrivateData.SubscriptionId,
	}

	// Build the channel metadata directly from the channel record so that the
	// adaptor can resolve URLs, headers, and keys without the original context.
	info.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelType:          channel.Type,
		ChannelId:            channel.Id,
		ChannelIsMultiKey:    channel.ChannelInfo.IsMultiKey,
		ChannelMultiKeyIndex: channel.ChannelInfo.MultiKeyPollingIndex,
		ChannelBaseUrl:       baseURL,
		ApiType:              apiType,
		ApiKey:               channel.Key,
		ChannelCreateTime:    channel.CreatedTime,
		ParamOverride:        make(map[string]interface{}),
		HeadersOverride:      make(map[string]interface{}),
		UpstreamModelName:    task.Properties.UpstreamModelName,
	}

	// Derive request URL path from the relay mode.
	switch info.RelayMode {
	case relayconstant.RelayModeImagesEdits:
		info.RequestURLPath = "/v1/images/edits"
	default:
		info.RequestURLPath = "/v1/images/generations"
	}

	// Reconstruct price data from the persisted billing context so that the
	// worker can settle the exact pre-consumed amount.
	if bc := task.PrivateData.BillingContext; bc != nil {
		info.PriceData = types.PriceData{
			ModelPrice:        bc.ModelPrice,
			ModelRatio:        bc.ModelRatio,
			UsePrice:          bc.PerCallBilling,
			Quota:             task.Quota,
			QuotaToPreConsume: task.Quota,
		}
		info.PriceData.GroupRatioInfo = types.GroupRatioInfo{
			GroupRatio: bc.GroupRatio,
		}
		if len(bc.OtherRatios) > 0 {
			info.PriceData.OtherRatios = make(map[string]float64, len(bc.OtherRatios))
			for k, v := range bc.OtherRatios {
				info.PriceData.OtherRatios[k] = v
			}
		}
	}

	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{
		Action:       constant.TaskActionImageGenerate,
		PublicTaskID: task.TaskID,
		ConsumeQuota: true,
	}

	return info, nil
}

// BuildFakeGinContext creates a minimal gin.Context from the worker context so
// that the existing adaptor pipeline (which expects *gin.Context) can be
// reused without modification. The returned context is bound to a response
// recorder that captures the upstream response body.
func BuildFakeGinContext(ctx context.Context, task *model.Task, requestBody []byte) (*gin.Context, *httptest.ResponseRecorder, error) {
	if task == nil {
		return nil, nil, fmt.Errorf("task is nil")
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	method := http.MethodPost
	switch task.PrivateData.RelayMode {
	case relayconstant.RelayModeImagesEdits:
		method = http.MethodPost
	default:
		method = http.MethodPost
	}

	req := httptest.NewRequest(method, "/v1/images/generations", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Restore the downstream-facing scheme/host so that image proxy URLs can be
	// rewritten to the same base URL the client used when submitting the task.
	if task.PrivateData.DownstreamBaseURL != "" {
		if u, err := url.Parse(task.PrivateData.DownstreamBaseURL); err == nil && u.Host != "" {
			req.Header.Set("X-Forwarded-Proto", u.Scheme)
			req.Header.Set("X-Forwarded-Host", u.Host)
			req.Host = u.Host
		}
	}

	req = req.WithContext(ctx)
	c.Request = req

	return c, recorder, nil
}

// CaptureImageResponse extracts the response body captured by the response
// recorder after the adaptor pipeline has executed. It returns the bytes written
// to the recorder or nil if nothing was written.
func CaptureImageResponse(recorder *httptest.ResponseRecorder) []byte {
	if recorder == nil {
		return nil
	}
	return recorder.Body.Bytes()
}
