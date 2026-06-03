# new-api 错误处理体系升级开发方案

> **基于**：《竞品错误处理最佳实践调研报告 2026-05-28》
> **日期**：2026-05-28
> **目标**：在保持 OpenAI 兼容性的前提下，全面提升错误响应的可读性、可操作性与可追踪性

---

## 一、现状诊断

### 1.1 当前架构

```
controller/relay.go ──► middleware/auth.go ──► relay/xxx_handler.go ──► service/error.go (RelayErrorHandler)
        │                                              │                         │
        ▼                                              ▼                         ▼
  abortWithOpenAiMessage()                    types.NewAPIError            dto.GeneralErrorResponse
  (gin.H 手写 JSON)                           (核心错误类型)                (上游错误解析)
```

### 1.2 现有能力

| 能力 | 现状 | 评价 |
|------|------|------|
| `NewAPIError` 结构 | `Err`, `RelayError`, `errorType`, `errorCode`, `StatusCode`, `Metadata` | 基础完善 |
| `ToOpenAIError()` | 支持 OpenAI/Claude 格式互转 | 架构正确 |
| `MaskSensitiveInfo()` | 内置正则脱敏 URL/IP/API Key | 生产级 |
| `ToMessage()` | 按优先级提取上游消息 | 够用但缺少可操作性 |
| `RequestID` | 通过 `common.MessageWithRequestId()` 拼接进 message | 不够结构化 |

### 1.3 核心差距

1. **缺少 `request_id` 结构化输出**：当前把 request_id 拼在 message 字符串里，不利于客户端提取和售后追踪。
2. **缺少 "What to do" 提示**：`ToMessage()` 只是提取上游消息，对内部错误（额度不足、渠道不可用等）没有标准用户指导。
3. **缺少 `Retry-After`**：429 响应没有标准退避头。
4. **缺少 `param` 透出**：参数校验错误不告诉用户哪个字段错了。
5. **异步任务错误结构不统一**：Task 失败没有统一格式。

---

## 二、总体目标

建立 **`ErrorCode → 话术模板 → 多语言 → 文档链接`** 的映射体系，核心交付物：

```
┌─────────────────────────────────────────────────────────┐
│  对外错误响应 (OpenAI 兼容 + 扩展)                        │
│  {                                                      │
│    "error": {                                           │
│      "message": "您的账户额度不足，请前往控制台充值...",    │  ← 给用户看 (actionable)
│      "type": "quota_error",                             │
│      "code": "insufficient_user_quota",                 │  ← snake_case
│      "param": "",                                       │  ← 参数校验时填充
│      "detail": "user_id=123, remaining_quota=0"         │  ← 给开发者看 (调试)
│      "help": "https://docs.newapi.com/errors/E1001"     │  ← 文档链接 (P1)
│      "provider_specific_fields": {...}                   │  ← 上游原始错误 (P1)
│    },                                                   │
│    "request_id": "req_abc123xyz"                        │  ← 结构化追踪
│    "retry_after": 60                                    │  ← 429 时存在 (P1)
│  }                                                      │
└─────────────────────────────────────────────────────────┘
```

---

## 三、阶段化开发方案

### 第一阶段：P0 — 错误响应增强 + 用户指导话术（Week 1）

**目标**：让每条错误都包含 `request_id` + `message`（用户友好）+ `detail`（调试），对内部错误附加标准指导话术。

#### 3.1.1 改造 `types/error.go` — `NewAPIError` 结构增强

**改动文件**：`types/error.go`

**新增字段**：

```go
type NewAPIError struct {
    Err            error
    RelayError     any
    skipRetry      bool
    recordErrorLog *bool
    errorType      ErrorType
    errorCode      ErrorCode
    StatusCode     int
    Metadata       json.RawMessage
    
    // === 新增字段 ===
    RequestID             string                 `json:"-"`  // 从 gin context 获取
    Param                 string                 `json:"-"`  // 出错参数名
    Detail                string                 `json:"-"`  // 调试详情（给开发者）
    ProviderSpecificFields map[string]interface{} `json:"-"`  // 上游原始错误结构
}
```

**新增方法**：

```go
// SetParam 设置出错参数名
func (e *NewAPIError) SetParam(param string) *NewAPIError {
    e.Param = param
    return e
}

// SetDetail 设置调试详情
func (e *NewAPIError) SetDetail(detail string) *NewAPIError {
    e.Detail = detail
    return e
}

// SetProviderSpecificFields 保留上游原始错误
func (e *NewAPIError) SetProviderSpecificFields(fields map[string]interface{}) *NewAPIError {
    e.ProviderSpecificFields = fields
    return e
}
```

#### 3.1.2 改造 `types/error.go` — `ToOpenAIError()` 输出增强

**改动目标**：对外输出时增加 `detail` 扩展字段，保持 OpenAI 核心字段兼容。

```go
type OpenAIError struct {
    Message  string          `json:"message"`
    Type     string          `json:"type"`
    Param    string          `json:"param"`
    Code     any             `json:"code"`
    Metadata json.RawMessage `json:"metadata,omitempty"`
    // === 新增扩展字段 ===
    Detail   string                 `json:"detail,omitempty"`    // 调试信息
    Help     string                 `json:"help,omitempty"`      // 文档链接 (P1)
    ProviderSpecificFields map[string]interface{} `json:"provider_specific_fields,omitempty"` // (P1)
}
```

```go
func (e *NewAPIError) ToOpenAIError() OpenAIError {
    var result OpenAIError
    switch e.errorType {
    case ErrorTypeOpenAIError:
        if openAIError, ok := e.RelayError.(OpenAIError); ok {
            result = openAIError
        }
    // ... 其他 case 不变 ...
    }
    
    // === 新增：填充扩展字段 ===
    result.Param = e.Param
    result.Detail = e.Detail
    result.ProviderSpecificFields = e.ProviderSpecificFields
    
    // === 新增：用模板替换内部错误消息 ===
    if e.errorType == ErrorTypeNewAPIError && e.errorCode != "" {
        template := GetErrorTemplate(e.errorCode, "zh") // 默认中文
        if template != "" {
            result.Message = RenderTemplate(template, e)
        }
    }
    
    // 脱敏 + fallback
    if e.errorCode != ErrorCodeCountTokenFailed {
        result.Message = common.MaskSensitiveInfo(result.Message)
    }
    if result.Message == "" {
        result.Message = string(e.errorType)
    }
    return result
}
```

#### 3.1.3 新建 `types/error_templates.go` — 用户指导话术映射表

**新建文件**：`types/error_templates.go`

```go
package types

import "strings"

// ErrorTemplate 单条错误的话术模板
type ErrorTemplate struct {
    Code     ErrorCode          // 错误码
    Type     string             // error.type 的值（领域分类）
    Message  map[string]string  // message 模板 {lang: template}
    Detail   map[string]string  // detail 模板 {lang: template}
    HelpURL  string             // 帮助文档链接
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
            "zh": "用户额度已耗尽。user_id={user_id}, remaining_quota={remaining_quota}",
            "en": "User quota exhausted. user_id={user_id}, remaining_quota={remaining_quota}",
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
        Code: ErrorCodeChannelModelMappedError,
        Type: "channel_error",
        Message: map[string]string{
            "zh": "模型 '{model}' 映射配置有误，请检查渠道设置中的模型映射规则。",
            "en": "Model '{model}' mapping configuration is incorrect. Please check the model mapping rules in channel settings.",
        },
        HelpURL: "https://docs.newapi.com/errors/channel/model-mapped",
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
        Code: ErrorCodeModelNotFound,
        Type: "invalid_request_error",
        Message: map[string]string{
            "zh": "模型 '{model}' 不可用，请检查模型名称是否正确，或切换至其他渠道。",
            "en": "Model '{model}' is not available. Please check the model name or switch to another channel.",
        },
        HelpURL: "https://docs.newapi.com/errors/request/model-not-found",
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
}

// errorTemplateMap 快速查找索引
var errorTemplateMap map[ErrorCode]*ErrorTemplate

func init() {
    errorTemplateMap = make(map[ErrorCode]*ErrorTemplate)
    for i := range errorTemplates {
        errorTemplateMap[errorTemplates[i].Code] = &errorTemplates[i]
    }
}

// GetErrorTemplate 获取错误模板
func GetErrorTemplate(code ErrorCode, lang string) *ErrorTemplate {
    tpl, ok := errorTemplateMap[code]
    if !ok {
        return nil
    }
    // 如果指定语言不存在，fallback 到英文
    if _, ok := tpl.Message[lang]; !ok {
        // 如果英文也没有，返回原始模板让用户自行 fallback
    }
    return tpl
}

// RenderTemplate 渲染模板，替换变量
func RenderTemplate(tpl *ErrorTemplate, err *NewAPIError, lang string) string {
    if tpl == nil {
        return err.Error()
    }
    msg := tpl.Message[lang]
    if msg == "" {
        msg = tpl.Message["en"]
    }
    if msg == "" {
        return err.Error()
    }
    
    // 变量替换
    msg = strings.ReplaceAll(msg, "{request_id}", err.RequestID)
    msg = strings.ReplaceAll(msg, "{status_code}", fmt.Sprintf("%d", err.StatusCode))
    msg = strings.ReplaceAll(msg, "{model}", err.Param) // 复用 Param 字段存模型名
    // 其他变量通过 Detail 字段的上下文提取
    
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
    return "new_api_error"
}
```

#### 3.1.4 改造 `middleware/utils.go` — 统一响应体增加 `request_id`

**改动文件**：`middleware/utils.go`

```go
func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...types.ErrorCode) {
    codeStr := ""
    if len(code) > 0 {
        codeStr = string(code[0])
    }
    userId := c.GetInt("id")
    requestId := c.GetString(common.RequestIdKey)
    
    // === 新增：构造结构化响应 ===
    resp := gin.H{
        "error": gin.H{
            "message": common.MessageWithRequestId(message, requestId),
            "type":    "new_api_error",
            "code":    codeStr,
        },
        "request_id": requestId,  // 结构化 request_id
    }
    
    // 如果有模板，用模板替换 message
    if len(code) > 0 {
        tpl := types.GetErrorTemplate(code[0], "zh")
        if tpl != nil {
            resp["error"].(gin.H)["message"] = tpl.Message["zh"]
            resp["error"].(gin.H)["type"] = tpl.Type
            if tpl.HelpURL != "" {
                resp["error"].(gin.H)["help"] = tpl.HelpURL
            }
        }
    }
    
    c.JSON(statusCode, resp)
    c.Abort()
    logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | request_id=%s | %s", userId, requestId, message))
}
```

#### 3.1.5 改造 `service/error.go` — `RelayErrorHandler` 增加 `ProviderSpecificFields`

**改动文件**：`service/error.go`

```go
func RelayErrorHandler(ctx context.Context, resp *http.Response, showBodyWhenFail bool) (newApiErr *types.NewAPIError) {
    newApiErr = types.InitOpenAIError(types.ErrorCodeBadResponseStatusCode, resp.StatusCode)
    // ... 现有解析逻辑 ...
    
    if common.GetJsonType(errResponse.Error) == "object" {
        oaiError := errResponse.TryToOpenAIError()
        if oaiError != nil {
            newApiErr = types.WithOpenAIError(*oaiError, resp.StatusCode)
            // === 新增：保留上游原始错误结构 ===
            var providerFields map[string]interface{}
            if err := common.Unmarshal(responseBody, &providerFields); err == nil {
                newApiErr.SetProviderSpecificFields(providerFields)
            }
            // ...
        }
    }
    // ...
    return
}
```

#### 3.1.6 验收标准

- [ ] 所有内部错误响应包含 `request_id` 字段
- [ ] 所有内部错误响应 `message` 为标准中文指导话术（非裸异常文本）
- [ ] 额度不足错误返回：`"您的账户额度不足，请前往控制台充值后再试。"`
- [ ] 渠道无可用 Key 错误返回：`"当前渠道 API Key 已耗尽或全部失效..."`
- [ ] 上游错误返回包含 `request_id` 用于售后追踪
- [ ] `error.type` 按领域分类（`quota_error`, `channel_error`, `auth_error` 等）

---

### 第二阶段：P1 — 错误码体系优化 + 基础设施完善（Week 2）

#### 3.2.1 429 `Retry-After` Header 支持

**改动文件**：`middleware/distributor.go`（或限流中间件）

```go
// 在返回 429 时增加 Retry-After 头
func rateLimitHandler(c *gin.Context) {
    c.Header("Retry-After", "60")  // 根据策略动态计算
    abortWithOpenAiMessage(c, http.StatusTooManyRequests, "Rate limit exceeded", types.ErrorCodePreConsumeTokenQuotaFailed)
}
```

**改动文件**：`service/error.go` — `RelayErrorHandler`

```go
// 上游返回 429 时，透传 Retry-After
if resp.StatusCode == http.StatusTooManyRequests {
    retryAfter := resp.Header.Get("Retry-After")
    if retryAfter != "" {
        // 存储到 NewAPIError 中，最终写入响应头
        // 需要扩展 NewAPIError 或增加 context 传递
    }
}
```

**新建文件**：`relay/common/retry_after.go` — 统一 Retry-After 管理

```go
package common

import (
    "context"
    "strconv"
    
    "github.com/gin-gonic/gin"
)

type retryAfterKey struct{}

func SetRetryAfter(ctx context.Context, seconds int) context.Context {
    return context.WithValue(ctx, retryAfterKey{}, seconds)
}

func GetRetryAfter(ctx context.Context) int {
    v, ok := ctx.Value(retryAfterKey{}).(int)
    if !ok {
        return 0
    }
    return v
}

func ApplyRetryAfterHeader(c *gin.Context, ctx context.Context) {
    seconds := GetRetryAfter(ctx)
    if seconds > 0 {
        c.Header("Retry-After", strconv.Itoa(seconds))
    }
}
```

#### 3.2.2 `param` 字段透出 — 参数校验类错误

**改动文件**：各 Handler 中的参数校验逻辑

示例（`relay/helper/valid_request.go`）：

```go
// 当参数校验失败时
return types.NewErrorWithStatusCode(
    fmt.Errorf("temperature must be between 0 and 2"),
    types.ErrorCodeBadRequestBody,
    http.StatusUnprocessableEntity,
).SetParam("temperature")
```

**改动文件**：`types/error.go` — `ToOpenAIError()`

```go
result.Param = e.Param  // 自动透出
```

#### 3.2.3 错误码前缀规范化（向后兼容）

**方案**：不改现有常量定义，在对外输出时做映射。

**新增文件**：`types/error_code_mapping.go`

```go
package types

import "strings"

// ToExternalCode 将内部 ErrorCode 映射为对外标准格式
func (code ErrorCode) ToExternalCode() string {
    s := string(code)
    // 将冒号分隔符替换为下划线
    s = strings.ReplaceAll(s, ":", "_")
    return s
}
```

在 `ToOpenAIError()` 中使用：

```go
result.Code = e.errorCode.ToExternalCode()
```

这样 `channel:no_available_key` → `channel_no_available_key`，`insufficient_user_quota` 保持不变。

#### 3.2.4 `help` 文档链接（P1）

在 `error_templates.go` 的每个模板中已定义 `HelpURL`，在 `ToOpenAIError()` 中自动填充：

```go
if tpl != nil && tpl.HelpURL != "" {
    result.Help = tpl.HelpURL
}
```

#### 3.2.5 `provider_specific_fields` 透传（P1）

在 `ToOpenAIError()` 中：

```go
if len(e.ProviderSpecificFields) > 0 {
    result.ProviderSpecificFields = e.ProviderSpecificFields
}
```

这样下游客户端可以访问上游原始错误结构，方便排查。

#### 3.2.6 验收标准

- [ ] 429 响应包含 `Retry-After` Header
- [ ] 参数校验错误（如 temperature 越界）返回 `param` 字段
- [ ] 对外 `code` 全部为 snake_case，无冒号
- [ ] 上游 429 透传 `Retry-After` 值
- [ ] 错误响应包含 `help` 文档链接
- [ ] 错误响应包含 `provider_specific_fields`（当上游返回结构化错误时）

---

### 第三阶段：P1/P2 — 多语言 + 异步任务 + Webhook（Week 3）

#### 3.3.1 多语言支持

**改动文件**：`types/error_templates.go` — `RenderTemplate()` 增加 `Accept-Language` 判断

```go
func GetLanguage(c *gin.Context) string {
    lang := c.GetHeader("Accept-Language")
    if strings.HasPrefix(lang, "zh") {
        return "zh"
    }
    return "en"
}

// 在 abortWithOpenAiMessage 中
lang := GetLanguage(c)
tpl := types.GetErrorTemplate(code[0], lang)
```

#### 3.3.2 异步任务错误结构统一

**改动文件**：`dto/task.go`（或相关 Task 结构定义文件）

```go
// TaskErrorResponse 统一异步任务错误响应
type TaskErrorResponse struct {
    TaskID    string      `json:"task_id"`
    Status    string      `json:"status"`   // "failed"
    Error     OpenAIError `json:"error"`    // 复用 OpenAIError 格式
    RequestID string      `json:"request_id"`
}
```

在 `service/task_polling.go` 等任务状态更新逻辑中：

```go
// 任务失败时构建统一错误响应
taskErr := TaskErrorResponse{
    TaskID:    task.ID,
    Status:    "failed",
    Error:     newAPIErr.ToOpenAIError(),
    RequestID: requestId,
}
```

#### 3.3.3 失败任务自动退款

**改动文件**：`service/task_polling.go`（或任务状态机）

```go
// 当任务状态从 running 变为 failed 时
if newStatus == "failed" && oldStatus != "failed" {
    // 自动回退预扣额度
    if err := RefundTaskQuota(task.ID); err != nil {
        logger.LogError(ctx, fmt.Sprintf("refund task quota failed: %v", err))
    }
}
```

#### 3.3.4 Webhook 失败隔离（P2）

**方案概述**：

1. Webhook 投递使用独立队列（内存队列或 Redis 队列）
2. 投递失败不影响任务终态
3. 指数退避重试：1min → 5min → 30min → 1h，最长 24h
4. 任务结果仍保留在轮询端点

**新建文件**：`service/webhook.go`

```go
package service

import (
    "context"
    "time"
)

type WebhookJob struct {
    URL         string
    Payload     []byte
    RetryCount  int
    NextRetryAt time.Time
}

var webhookQueue = make(chan WebhookJob, 1000)

func StartWebhookWorker(ctx context.Context) {
    go func() {
        for job := range webhookQueue {
            // 执行投递
            success := deliverWebhook(ctx, job)
            if !success && job.RetryCount < 4 {
                job.RetryCount++
                job.NextRetryAt = time.Now().Add(getBackoffDuration(job.RetryCount))
                // 重新入队（实际实现可用 Redis/DB 持久化）
                go func(j WebhookJob) {
                    time.Sleep(time.Until(j.NextRetryAt))
                    webhookQueue <- j
                }(job)
            }
        }
    }()
}

func getBackoffDuration(retryCount int) time.Duration {
    switch retryCount {
    case 1: return 1 * time.Minute
    case 2: return 5 * time.Minute
    case 3: return 30 * time.Minute
    default: return 1 * time.Hour
    }
}
```

#### 3.3.5 验收标准

- [ ] `Accept-Language: en` 返回英文错误消息
- [ ] 异步任务失败返回统一 `TaskErrorResponse` 格式
- [ ] 异步任务失败自动退还预扣额度
- [ ] Webhook 投递失败不阻塞任务终态

---

## 四、改造后完整响应示例对比

### 4.1 改造前（当前）

```json
{
  "error": {
    "message": "insufficient_user_quota",
    "type": "new_api_error",
    "code": "insufficient_user_quota"
  }
}
```

### 4.2 改造后（P0 完成）

```json
{
  "error": {
    "message": "您的账户额度不足，请前往控制台充值后再试。",
    "type": "quota_error",
    "code": "insufficient_user_quota",
    "detail": "user_id=123, remaining_quota=0"
  },
  "request_id": "req_abc123xyz"
}
```

### 4.3 改造后（P1 完成）

```json
{
  "error": {
    "message": "您的账户额度不足，请前往控制台充值后再试。",
    "type": "quota_error",
    "code": "insufficient_user_quota",
    "param": "",
    "detail": "user_id=123, remaining_quota=0",
    "help": "https://docs.newapi.com/errors/quota/insufficient",
    "provider_specific_fields": {
      "upstream_body": "{...}"
    }
  },
  "request_id": "req_abc123xyz",
  "retry_after": 60
}
```

---

## 五、HTTP 状态码优化对照表

| 场景 | 当前 | 建议 | 改动位置 |
|------|------|------|----------|
| JSON 解析失败 | 400 | 400 | 已有 |
| 参数值越界（temperature=2.5） | 400 | **422** | `relay/helper/valid_request.go` |
| 模型不存在 | 400 | 404 | `relay/xxx_handler.go` |
| API Key 无效 | 400 | **401** | `middleware/auth.go` |
| 无权限访问模型 | 400 | **403** | `middleware/auth.go` |
| 请求体过大 | 400 | **413** | 中间件 |
| 额度不足 | 429 | **429** + Retry-After | `service/billing.go` |
| 渠道无可用 Key | 500 | **503** | `service/channel.go` |
| 上游超时 | 500 | **502/504** | `service/error.go` |
| 上游返回畸形数据 | 500 | **502** | `service/error.go` |

---

## 六、改动文件清单

### P0（Week 1）

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `types/error.go` | 修改 | `NewAPIError` 增加 `RequestID`, `Param`, `Detail`, `ProviderSpecificFields`；`ToOpenAIError()` 增加模板渲染 |
| `types/error_templates.go` | **新增** | 用户指导话术映射表（中英文） |
| `middleware/utils.go` | 修改 | `abortWithOpenAiMessage()` 输出 `request_id` + 模板消息 |
| `service/error.go` | 修改 | `RelayErrorHandler()` 保留上游原始错误结构 |
| `types/error_code_mapping.go` | **新增** | 对外错误码格式映射（冒号→下划线） |

### P1（Week 2）

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `relay/common/retry_after.go` | **新增** | `Retry-After` context 传递工具 |
| `middleware/distributor.go` | 修改 | 429 响应增加 `Retry-After` Header |
| `service/error.go` | 修改 | 上游 429 透传 `Retry-After` |
| `relay/helper/valid_request.go` | 修改 | 参数校验错误设置 `Param` 字段 |
| 各 `relay/xxx_handler.go` | 修改 | 模型不存在返回 404，权限不足返回 403 等 |
| `middleware/auth.go` | 修改 | Key 无效返回 401 |

### P2（Week 3）

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `types/error_templates.go` | 修改 | `RenderTemplate()` 增加多语言支持 |
| `dto/task.go` | 修改 | 增加 `TaskErrorResponse` 统一结构 |
| `service/task_polling.go` | 修改 | 任务失败时构建 `TaskErrorResponse`，自动退款 |
| `service/webhook.go` | **新增** | Webhook 独立重试队列 |
| `main.go` | 修改 | 启动 Webhook Worker |

---

## 七、风险与回滚策略

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 模板消息覆盖上游消息，丢失上下文 | 中 | 中 | 仅对 `ErrorTypeNewAPIError` 使用模板，上游错误保持透传 |
| 新增字段导致下游客户端解析失败 | 低 | 中 | 新增字段均为 `omitempty`，且放在 `error` 对象内，不破坏顶层结构 |
| `request_id` 格式变化影响现有日志分析 | 低 | 低 | 保持现有 `common.RequestIdKey` 格式不变，仅增加独立字段输出 |
| 多语言切换导致测试不稳定 | 低 | 低 | 默认中文，英文通过 `Accept-Language` 显式触发 |
| 状态码调整影响客户端重试逻辑 | 中 | 高 | **P0 不动状态码**，P1 单独做状态码调整，充分测试后上线 |

### 回滚策略

1. **配置开关**：在 `common` 包中增加 `EnableStructuredErrorResponse` 全局开关，默认 `false`（灰度），确认稳定后切 `true`。
2. **分阶段上线**：
   - Day 1：仅增加 `request_id` 字段（无风险）
   - Day 3：开启模板消息（仅内部错误）
   - Day 7：开启 `detail` + `help`
   - Day 14：开启状态码调整

---

## 八、与竞品格式对比（改造后）

| 维度 | OpenAI | Claude | **new-api（改造后）** |
|------|--------|--------|----------------------|
| `request_id` | Header only | Header + Body | **Body 结构化 + Header** |
| `error.message` | 人类可读 | 人类可读 | **人类可读 + 可操作** |
| `error.code` | snake_case | 嵌套 type | **snake_case** |
| `error.type` | 领域分类 | 嵌套两层 | **领域分类（quota/channel/auth）** |
| `error.param` | 有 | 无 | **有** |
| `error.detail` | 无 | 无 | **有（调试信息）** |
| `error.help` | 无 | 无 | **有（文档链接）** |
| `Retry-After` | 有 | 有 | **有** |
| 多语言 | 英文 | 英文 | **zh/en 切换** |
| 异步错误 | 无 | 无 | **统一 TaskErrorResponse** |
