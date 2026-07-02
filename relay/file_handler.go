package relay

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func FileHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	fileReq, ok := info.Request.(*dto.FileUploadRequest)
	if !ok {
		return types.NewErrorWithStatusCode(errors.New("invalid request type, expected dto.FileUploadRequest"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(fileReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to FileUploadRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	fileUploadAdaptor, ok := adaptor.(channel.FileUploadAdaptor)
	if !ok {
		return types.NewErrorWithStatusCode(errors.New("file upload is not supported by this channel"), types.ErrorCodeInvalidApiType, http.StatusNotImplemented, types.ErrOptionWithSkipRetry())
	}

	convertedRequest, err := fileUploadAdaptor.ConvertFileRequest(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}

	var requestBody io.Reader
	switch body := convertedRequest.(type) {
	case *bytes.Buffer:
		requestBody = body
	case io.Reader:
		requestBody = body
	default:
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
			statusCodeMappingStr := c.GetString("status_code_mapping")
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := fileUploadAdaptor.DoFileResponse(c, httpResp, info)
	if newAPIError != nil {
		statusCodeMappingStr := c.GetString("status_code_mapping")
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	if usage != nil {
		service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), nil)
	}

	return nil
}

// FileOperationHelper handles GET / DELETE file operations (list, retrieve, delete)
// by forwarding the request to the upstream channel and delegating response handling
// to the channel adaptor.
func FileOperationHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	resp, err := adaptor.DoRequest(c, info, nil)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
			statusCodeMappingStr := c.GetString("status_code_mapping")
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		statusCodeMappingStr := c.GetString("status_code_mapping")
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	if usage != nil {
		service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), nil)
	}

	return nil
}
