package types

import (
	"fmt"
	"strings"
)

// ErrorTemplate 单条错误的话术模板
type ErrorTemplate struct {
	Code    ErrorCode         // 错误码
	Type    string            // error.type 的值（领域分类）
	Message map[string]string // message 模板 {lang: template}
	Detail  map[string]string // detail 模板 {lang: template}
	HelpURL string            // 帮助文档链接
}

// 模板变量支持：
// {remaining_quota}  - 剩余额度
// {retry_after}      - 建议重试秒数
// {model}            - 模型名
// {param}            - 出错参数
// {request_id}       - 请求ID
// {status_code}      - HTTP 状态码
// {max_tokens}       - 最大上下文长度
// {timeout}          - 超时时间
// {reason}           - 失败原因
// {user_id}          - 用户ID

var errorTemplates = []ErrorTemplate{
	// === auth_xxx ===
	{
		Code: ErrorCodeAccessDenied,
		Type: "auth_error",
		Message: map[string]string{
			"zh": "您的账户没有权限访问此模型或服务，请联系管理员开通权限。",
			"en": "Your account does not have permission to access this model or service. Please contact your administrator.",
		},
		HelpURL: "https://docs.newapi.com/errors/auth/access-denied",
	},
	{
		Code: ErrorCodeChannelInvalidKey,
		Type: "auth_error",
		Message: map[string]string{
			"zh": "API Key 无效或已过期，请检查您的密钥设置，或前往控制台重新生成。",
			"en": "Invalid API key or key expired. Please check your API key settings or generate a new one in the console.",
		},
		HelpURL: "https://docs.newapi.com/errors/auth/invalid-key",
	},

	// === quota_xxx ===
	{
		Code: ErrorCodeInsufficientUserQuota,
		Type: "quota_error",
		Message: map[string]string{
			"zh": "您的账户额度不足，请前往控制台充值后再试。",
			"en": "Insufficient account quota. Please recharge in the console and try again.",
		},
		Detail: map[string]string{
			"zh": "用户额度已耗尽。request_id={request_id}, status_code={status_code}",
			"en": "User quota exhausted. request_id={request_id}, status_code={status_code}",
		},
		HelpURL: "https://docs.newapi.com/errors/quota/insufficient",
	},
	{
		Code: ErrorCodePreConsumeTokenQuotaFailed,
		Type: "quota_error",
		Message: map[string]string{
			"zh": "预扣额度失败，可能是额度不足或并发冲突，请稍后重试。",
			"en": "Failed to pre-consume quota. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/quota/pre-consume-failed",
	},

	// === channel_xxx ===
	{
		Code: ErrorCodeChannelNoAvailableKey,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "当前渠道 API Key 已耗尽或全部失效，请联系管理员添加新的 API Key。",
			"en": "All API keys for the current channel are exhausted or invalid. Please contact your administrator to add new API keys.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/no-available-key",
	},
	{
		Code: ErrorCodeChannelParamOverrideInvalid,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "渠道参数覆盖配置无效，请检查渠道设置中的参数覆盖规则。",
			"en": "Channel parameter override configuration is invalid. Please check the parameter override rules in channel settings.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/param-override",
	},
	{
		Code: ErrorCodeChannelHeaderOverrideInvalid,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "渠道请求头覆盖配置无效，请检查渠道设置中的请求头覆盖规则。",
			"en": "Channel header override configuration is invalid. Please check the header override rules in channel settings.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/header-override",
	},
	{
		Code: ErrorCodeChannelModelMappedError,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "模型映射配置有误，请检查渠道设置中的模型映射规则。",
			"en": "Model mapping configuration is incorrect. Please check the model mapping rules in channel settings.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/model-mapped",
	},
	{
		Code: ErrorCodeChannelAwsClientError,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "AWS 客户端初始化失败，请检查渠道 AWS 配置。",
			"en": "AWS client initialization failed. Please check the channel AWS configuration.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/aws-client",
	},
	{
		Code: ErrorCodeChannelResponseTimeExceeded,
		Type: "channel_error",
		Message: map[string]string{
			"zh": "渠道响应时间超限，请稍后重试或联系管理员。",
			"en": "Channel response time exceeded. Please retry later or contact your administrator.",
		},
		HelpURL: "https://docs.newapi.com/errors/channel/response-time",
	},

	// === request_xxx ===
	{
		Code: ErrorCodeBadRequestBody,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "请求参数有误，请检查请求体格式是否符合要求。",
			"en": "Invalid request parameters. Please check that the request body format meets the requirements.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/bad-request",
	},
	{
		Code: ErrorCodeReadRequestBodyFailed,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "读取请求体失败，请检查请求是否完整。",
			"en": "Failed to read request body. Please check that the request is complete.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/read-body-failed",
	},
	{
		Code: ErrorCodeConvertRequestFailed,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "请求转换失败，请检查请求参数格式。",
			"en": "Request conversion failed. Please check the request parameter format.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/convert-failed",
	},
	{
		Code: ErrorCodeModelNotFound,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "模型不可用，请检查模型名称是否正确，或切换至其他渠道。",
			"en": "Model is not available. Please check the model name or switch to another channel.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/model-not-found",
	},
	{
		Code: ErrorCodeInvalidRequest,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "请求无效，请检查请求参数。",
			"en": "Invalid request. Please check the request parameters.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/invalid",
	},
	{
		Code: ErrorCodePromptBlocked,
		Type: "content_policy_violation",
		Message: map[string]string{
			"zh": "输入内容触发了安全策略，请修改后重试。如涉及误判，请联系客服申诉。",
			"en": "Input violates content safety policy. Please revise and retry. Contact support if you believe this is a false positive.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/content-policy",
	},
	{
		Code: ErrorCodeSensitiveWordsDetected,
		Type: "content_policy_violation",
		Message: map[string]string{
			"zh": "输入内容包含敏感词，请修改后重试。",
			"en": "Input contains sensitive words. Please revise and retry.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/sensitive-words",
	},

	// === server_xxx ===
	{
		Code: ErrorCodeBadResponseStatusCode,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "上游服务暂时异常，请稍后重试。如问题持续，请提供 Request ID '{request_id}' 联系支持团队。",
			"en": "Upstream service temporarily unavailable. Please retry later. If the issue persists, contact support with Request ID '{request_id}'.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/upstream",
	},
	{
		Code: ErrorCodeDoRequestFailed,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "请求上游服务失败，请稍后重试。",
			"en": "Failed to request upstream service. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/request-failed",
	},
	{
		Code: ErrorCodeGetChannelFailed,
		Type: "server_error",
		Message: map[string]string{
			"zh": "获取渠道信息失败，请稍后重试或联系管理员。",
			"en": "Failed to get channel information. Please retry later or contact your administrator.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/channel-failed",
	},
	{
		Code: ErrorCodeGenRelayInfoFailed,
		Type: "server_error",
		Message: map[string]string{
			"zh": "生成中继信息失败，请稍后重试。",
			"en": "Failed to generate relay information. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/relay-info",
	},
	{
		Code: ErrorCodeReadResponseBodyFailed,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "读取上游响应失败，请稍后重试。",
			"en": "Failed to read upstream response. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/read-response",
	},
	{
		Code: ErrorCodeBadResponse,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "上游响应异常，请稍后重试。",
			"en": "Upstream response abnormal. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/bad-response",
	},
	{
		Code: ErrorCodeEmptyResponse,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "上游返回空响应，请稍后重试。",
			"en": "Upstream returned empty response. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/empty-response",
	},
	{
		Code: ErrorCodeAwsInvokeError,
		Type: "upstream_error",
		Message: map[string]string{
			"zh": "AWS 调用失败，请检查 AWS 配置或稍后重试。",
			"en": "AWS invocation failed. Please check AWS configuration or retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/aws-invoke",
	},

	// === sql / data error ===
	{
		Code: ErrorCodeQueryDataError,
		Type: "server_error",
		Message: map[string]string{
			"zh": "数据查询失败，请稍后重试。",
			"en": "Data query failed. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/query-data",
	},
	{
		Code: ErrorCodeUpdateDataError,
		Type: "server_error",
		Message: map[string]string{
			"zh": "数据更新失败，请稍后重试。",
			"en": "Data update failed. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/update-data",
	},

	// === internal ===
	{
		Code: ErrorCodeCountTokenFailed,
		Type: "server_error",
		Message: map[string]string{
			"zh": "Token 计算失败，请稍后重试。",
			"en": "Token counting failed. Please retry later.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/count-token",
	},
	{
		Code: ErrorCodeModelPriceError,
		Type: "server_error",
		Message: map[string]string{
			"zh": "模型价格配置错误，请联系管理员。",
			"en": "Model price configuration error. Please contact your administrator.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/model-price",
	},
	{
		Code: ErrorCodeInvalidApiType,
		Type: "invalid_request_error",
		Message: map[string]string{
			"zh": "无效的 API 类型，请检查请求路径。",
			"en": "Invalid API type. Please check the request path.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/invalid-api-type",
	},
	{
		Code: ErrorCodeJsonMarshalFailed,
		Type: "server_error",
		Message: map[string]string{
			"zh": "JSON 序列化失败，请检查请求数据格式。",
			"en": "JSON serialization failed. Please check the request data format.",
		},
		HelpURL: "https://docs.newapi.com/errors/server/json-marshal",
	},
	{
		Code: ErrorCodeViolationFeeGrokCSAM,
		Type: "content_policy_violation",
		Message: map[string]string{
			"zh": "内容触发安全策略违规，已产生违规费用。请修改输入内容后重试。",
			"en": "Content triggered a safety policy violation and incurred a violation fee. Please revise your input and retry.",
		},
		HelpURL: "https://docs.newapi.com/errors/request/violation-fee",
	},
}

// errorTemplateMap 快速查找索引
var errorTemplateMap map[ErrorCode]*ErrorTemplate

func init() {
	errorTemplateMap = make(map[ErrorCode]*ErrorTemplate)
	for i := range errorTemplates {
		errorTemplateMap[errorTemplates[i].Code] = &errorTemplates[i]
	}
}

// GetErrorTemplate 获取错误模板（不指定语言，返回原始模板）
func GetErrorTemplate(code ErrorCode) *ErrorTemplate {
	return errorTemplateMap[code]
}

// RenderTemplate 渲染模板，替换变量
func RenderTemplate(tpl *ErrorTemplate, err *NewAPIError, lang string) string {
	if tpl == nil {
		return ""
	}
	msg := tpl.Message[lang]
	if msg == "" {
		msg = tpl.Message["en"]
	}
	if msg == "" {
		return ""
	}

	// 变量替换
	if err != nil {
		msg = strings.ReplaceAll(msg, "{request_id}", err.RequestID)
		msg = strings.ReplaceAll(msg, "{status_code}", fmt.Sprintf("%d", err.StatusCode))
		msg = strings.ReplaceAll(msg, "{param}", err.Param)
		msg = strings.ReplaceAll(msg, "{model}", err.Param) // 复用 Param 字段存模型名
		msg = strings.ReplaceAll(msg, "{detail}", err.Detail)
	}
	return msg
}

// RenderDetailTemplate 渲染 detail 模板
func RenderDetailTemplate(tpl *ErrorTemplate, err *NewAPIError, lang string) string {
	if tpl == nil || tpl.Detail == nil {
		return ""
	}
	msg := tpl.Detail[lang]
	if msg == "" {
		msg = tpl.Detail["en"]
	}
	if msg == "" {
		return ""
	}

	if err != nil {
		msg = strings.ReplaceAll(msg, "{request_id}", err.RequestID)
		msg = strings.ReplaceAll(msg, "{status_code}", fmt.Sprintf("%d", err.StatusCode))
		msg = strings.ReplaceAll(msg, "{param}", err.Param)
		msg = strings.ReplaceAll(msg, "{model}", err.Param)
		msg = strings.ReplaceAll(msg, "{detail}", err.Detail)
	}
	return msg
}

// GetErrorTypeForCode 根据 ErrorCode 返回领域分类 type
func GetErrorTypeForCode(code ErrorCode) string {
	tpl := errorTemplateMap[code]
	if tpl != nil {
		return tpl.Type
	}
	// fallback 按前缀推断
	codeStr := string(code)
	if strings.HasPrefix(codeStr, "channel:") {
		return "channel_error"
	}
	if strings.HasPrefix(codeStr, "insufficient_") || strings.HasPrefix(codeStr, "pre_consume") {
		return "quota_error"
	}
	if strings.HasPrefix(codeStr, "invalid_") || strings.HasPrefix(codeStr, "bad_") || strings.HasPrefix(codeStr, "read_") {
		return "invalid_request_error"
	}
	if strings.HasPrefix(codeStr, "access_") {
		return "auth_error"
	}
	return "new_api_error"
}

// HasTemplate 判断错误码是否有模板
func HasTemplate(code ErrorCode) bool {
	_, ok := errorTemplateMap[code]
	return ok
}
