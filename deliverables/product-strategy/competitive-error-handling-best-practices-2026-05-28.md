# 竞品错误处理最佳实践调研报告

**日期**：2026-05-28  
**调研者**：竞析（Competitive Analyst）  
**覆盖范围**：OpenAI API、Anthropic Claude API、Google Gemini API、Azure OpenAI、LiteLLM/One API、Midjourney/Suno 异步模式

---

## 一、竞品错误响应格式对比表

| 维度 | OpenAI API | Anthropic Claude API | Google Gemini API | Azure OpenAI | LiteLLM (网关) | new-api (当前) |
|------|-----------|---------------------|-------------------|-------------|---------------|---------------|
| **顶层结构** | `{"error": {...}}` | `{"type":"error","error":{...},"request_id":"..."}` | Google Cloud Error Model: `{"error":{"code":400,"message":"...","status":"INVALID_ARGUMENT"}}` | 同 OpenAI 格式 | `{"error":{"message":"...","type":null,"param":null,"code":"400","provider_specific_fields":{...}}}` | `NewAPIError` → 可转多格式 |
| **error.type** | `invalid_request_error` / `rate_limit_error` 等 | 嵌套两层：`type:"error"` + `error.type:"rate_limit_error"` | 使用 `status` 字段：`INVALID_ARGUMENT`、`RESOURCE_EXHAUSTED` | 同 OpenAI | 继承 OpenAI 类型，额外有 `llm_provider` | `ErrorTypeOpenAIError` / `ErrorTypeClaudeError` / `ErrorTypeUpstreamError` 等 |
| **error.code** | `rate_limit_exceeded`、`context_length_exceeded`（snake_case） | 无独立 code，用 `error.type` 区分 | `code` 为 HTTP 状态码数字 | 同 OpenAI | 字符串化 HTTP 状态码 `"400"` | `ErrorCodeInsufficientUserQuota` / `ErrorCodeChannelNoAvailableKey` 等（细粒度字符串常量） |
| **error.message** | 人类可读，含具体上下文（如 "Limit 30000, Used 28000, Requested 5000"） | 简洁描述（如 "The requested resource could not be found."） | 人类可读描述 | 同 OpenAI | 聚合上游消息 + 提供商前缀 | 通过 `ToMessage()` 按优先级提取上游消息 |
| **error.param** | 有，标识出错参数 | 无 | 无（在 details 数组中） | 有 | 保留字段但常为 null | 未明确透传 |
| **request_id** | 响应体中无，但在 Header 中 | 响应体顶层 + Header `request-id` | 响应体无，通过 Cloud Logging 追踪 | 响应体无，通过 `x-ms-request-id` Header | 无 | 未明确提及 |
| **Retry-After** | 429 响应头必带 | 429 响应头带 | 429 建议指数退避 | 带 | 透传上游 | 未明确 |
| **多错误返回** | 单错误 | 单错误 | `details[]` 数组可含多维度信息 | 单错误 | 单错误 + `provider_specific_fields` | 单错误 |
| **敏感信息屏蔽** | 消息中不暴露堆栈 | 不暴露堆栈 | 不暴露堆栈 | 不暴露堆栈 | 不暴露堆栈 | 内置正则脱敏 URL/IP/API Key |

**关键发现**：
- **OpenAI** 是事实标准，几乎所有网关（LiteLLM、One API）都向其格式对齐。
- **Claude** 的嵌套 `error.type` + `request_id` 设计在调试体验上最佳。
- **Gemini** 遵循 Google Cloud 标准，使用 `status` + `details[]`，但可读性较弱。
- **LiteLLM** 的 `provider_specific_fields` 是优秀设计，保留了上游特有信息供高级用户排查。
- **new-api** 的 `NewAPIError` + `ErrorCode` 体系已非常完善，但在**用户友好度**和**可操作提示**上仍有提升空间。

---

## 二、最佳实践总结

### 2.1 错误提示的"黄金法则"

| 原则 | 说明 | 正反例 |
|------|------|--------|
| **清晰 (Clear)** | 用户一眼能懂问题所在 | "Rate limit reached for gpt-4o on tokens per min: Limit 30000, Used 28000" / "Error 429" |
| **可操作 (Actionable)** | 告诉用户下一步该做什么 | "Please check your API key or generate a new one at platform.openai.com" / "Authentication failed" |
| **不暴露敏感信息** | 隐藏内部拓扑、堆栈、密钥 | "Internal server error. Please try again later." / "connection refused to postgres://user:pass@10.0.1.5:5432" |
| **结构化机器可读** | 提供 `code` + `type` 供程序判断 | `code: "rate_limit_exceeded"` / 仅返回文本 "too many requests" |
| **可追踪** | 提供 `request_id` 用于售后排查 | Header `request-id: req_xxx` / 无任何追踪标识 |

### 2.2 HTTP 状态码使用建议

| 状态码 | 使用场景 | AI API 典型场景 |
|--------|----------|----------------|
| **400** | 请求格式错误、缺少必填参数、JSON 解析失败 | 模型名拼写错误、messages 格式非法、user_id 字段违规 |
| **401** | 认证信息缺失或无效（API Key 错误/过期） | Key 不存在、Key 被吊销、组织 ID 不匹配 |
| **403** | 已认证但无权限 | 模型访问权限不足、区域限制、IP 白名单拦截 |
| **404** | 资源不存在 | 模型不存在、File ID 不存在 |
| **413** | 请求体过大 | 文件超过 32MB/500MB 限制 |
| **422** | 请求格式正确但语义无法处理（如验证失败） | 参数组合非法（如 temperature=2.5）、结构化输出 Schema 冲突 |
| **429** | 速率限制触发 | RPM/TPM 超限、日配额用尽、账户余额不足 |
| **500** | 服务端未预期异常 | 上游内部错误、网关未知异常 |
| **502** | 网关从上游收到无效响应 | 上游返回畸形数据、上游连接中断 |
| **503** | 服务暂时不可用（维护或过载） | 模型过载、区域服务维护、渠道下线 |
| **529** | Claude 特有：模型过载 | 等同于 503 处理 |

**特别建议**：
- **400 vs 422**: 400 用于"语法错误"（如 JSON 格式不对），422 用于"语义错误"（如参数值超出范围）。多数 AI API 混用两者，但区分后能大幅降低用户排查成本。
- **429 必须带 Retry-After**: OpenAI 和 Claude 均会在响应头中返回 `Retry-After`（秒数），这是生产级客户端实现指数退避的关键依赖。
- **503 带维护窗口**: 若服务有计划维护，响应体中可附加 `retry_after` 字段或 `maintenance_window` 信息，避免用户盲目重试。

### 2.3 异步任务错误处理建议

| 阶段 | 最佳实践 |
|------|----------|
| **任务提交失败** | 立即同步返回 4xx 错误（参数校验、权限、额度），失败任务不计费 |
| **任务执行失败** | 通过 Webhook 推送失败事件：`{"event":"task.failed","task_id":"...","status":"failed","error":{"message":"...","type":"..."}}` |
| **Webhook 投递失败** | 与任务失败解耦，独立指数退避重试（1min → 5min → 30min → 1h，最长 24h），任务结果仍保留在轮询端点 |
| **超时处理** | 设置硬超时（如 1 小时），超时后状态置为 `failed`，错误类型固定为 `TimeoutError` |
| **计费策略** | 任务提交失败不计费；任务执行失败建议**全额退款**或**不计费**（Midjourney 模式）；部分完成场景按实际进度计费需提前声明 |
| **状态轮询** | 对 `pending`/`running` 任务返回 `Retry-After` 头，建议根据进度动态调整（进度>80% 时 5s，否则 15s） |

### 2.4 用户指导动作的标准模板

每条错误应包含三个层次的信息：

1. 发生了什么（What happened）
2. 为什么发生（Why it happened）
3. 用户该怎么做（What to do）

示例（OpenAI 429 风格）：
```json
{
  "error": {
    "message": "Rate limit reached for gpt-4o on tokens per min (TPM): Limit 30000, Used 28000, Requested 5000. Please try again in 20s.",
    "type": "tokens",
    "code": "rate_limit_exceeded"
  }
}
```

---

## 三、new-api 可借鉴的改进点

### 3.1 错误响应格式增强

| 优先级 | 改进点 | 具体动作 |
|--------|--------|----------|
| **P0** | **增加 `request_id`** | 在错误响应体顶层或 Header 中增加 `request-id`，便于用户反馈时快速定位。参考 Claude 的 `req_xxx` 格式 |
| **P0** | **统一错误消息中的 actionable 提示** | 当前 `ToMessage()` 仅提取上游消息，建议对内部错误（如额度不足、渠道不可用）附加固定话术模板 |
| **P1** | **增加 `help` 文档链接** | 对每个 `ErrorCode` 配置对应的文档链接，在响应体中返回 |
| **P1** | **429 响应增加 `Retry-After` Header** | 对上游 429 和内部限流，均透传或计算返回 `Retry-After` 头 |
| **P1** | **考虑 `provider_specific_fields` 透传** | 参考 LiteLLM，在错误响应中保留上游原始错误结构，供高级用户排查 |

### 3.2 错误码体系优化

| 优先级 | 改进点 | 具体动作 |
|--------|--------|----------|
| **P0** | **明确 `code` 的命名规范** | 当前 `ErrorCode` 是 Go 常量（如 `ErrorCodeInsufficientUserQuota`），建议对外暴露为 `snake_case`（如 `insufficient_user_quota`），与 OpenAI/Claude 保持一致 |
| **P1** | **按领域分类错误码前缀** | 如 `auth_xxx`、`quota_xxx`、`channel_xxx`、`model_xxx`，方便客户端按前缀做错误分类处理 |
| **P1** | **增加 `param` 字段透出** | 对参数校验类错误（如模型名错误、 temperature 越界），在响应中明确返回 `param` 字段标识出错参数 |

### 3.3 用户指导动作设计

| 优先级 | 改进点 | 具体动作 |
|--------|--------|----------|
| **P0** | **建立用户指导话术映射表** | 将每个 `ErrorCode` 映射到标准用户提示话术，确保不同渠道返回一致的用户体验 |
| **P0** | **区分"用户消息"和"调试详情"** | 当前错误消息直接透传上游。建议内部错误响应包含 `message`（给用户）+ `detail`（给开发者/管理员）双字段 |
| **P1** | **多语言支持** | 错误消息模板支持 i18n，根据 `Accept-Language` 或用户配置返回中英文提示 |
| **P1** | **前端友好的错误代码** | 对 Web UI 用户，返回带错误码的弹窗，附带"复制错误信息"和"查看帮助文档"按钮 |

### 3.4 异步任务改进

| 优先级 | 改进点 | 具体动作 |
|--------|--------|----------|
| **P1** | **统一异步任务错误结构** | Midjourney/Claude Batches 等异步场景，统一返回 `{"task_id":"...","status":"failed","error":{...}}` |
| **P1** | **失败任务自动退款** | 对异步任务（如 Midjourney 生成失败），状态置为 `failed` 时自动回退预扣额度 |
| **P2** | **Webhook 失败隔离** | Webhook 投递失败不影响任务终态，独立重试队列 |

---

## 四、用户指导话术模板库

### 4.1 认证/权限类 (auth_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `invalid_api_key` | "API Key 无效或已过期，请检查您的密钥设置，或前往控制台重新生成。" | "Invalid API key or key expired. Please check your API key settings or generate a new one in the console." |
| `access_denied` | "您的账户没有权限访问此模型或服务，请联系管理员开通权限。" | "Your account does not have permission to access this model or service. Please contact your administrator." |
| `ip_not_allowed` | "当前 IP 地址不在白名单内，请从授权的网络环境访问，或更新 IP 白名单设置。" | "Current IP address is not in the allowlist. Please access from an authorized network or update your IP allowlist settings." |

### 4.2 额度/计费类 (quota_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `insufficient_user_quota` | "您的账户额度不足，请前往控制台充值后再试。当前可用额度：{remaining_quota}。" | "Insufficient account quota. Please recharge in the console and try again. Current available quota: {remaining_quota}." |
| `rate_limit_exceeded` | "请求过于频繁，已触发速率限制。请 {retry_after} 秒后再试，或降低请求频率。" | "Rate limit exceeded. Please retry after {retry_after} seconds or reduce your request frequency." |
| `quota_exhausted` | "您的月度/日度配额已用完，如需继续使用请升级套餐或购买额外额度。" | "Your monthly/daily quota has been exhausted. Please upgrade your plan or purchase additional quota to continue." |

### 4.3 渠道/路由类 (channel_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `channel_unavailable` | "当前渠道暂不可用，系统正在自动切换至备用渠道，请稍后再试。" | "The current channel is temporarily unavailable. The system is automatically switching to a backup channel. Please try again shortly." |
| `channel_no_available_key` | "当前渠道 API Key 已耗尽或全部失效，请联系管理员添加新的 API Key。" | "All API keys for the current channel are exhausted or invalid. Please contact your administrator to add new API keys." |
| `model_not_found` | "模型 '{model}' 在当前渠道不可用，请检查模型名称是否正确，或切换至其他渠道。" | "Model '{model}' is not available on the current channel. Please check the model name or switch to another channel." |
| `region_not_supported` | "该模型在您的所在区域暂不可用，请选择其他模型或切换区域。" | "This model is not available in your region. Please select another model or switch regions." |

### 4.4 请求参数类 (request_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `invalid_request_error` | "请求参数有误，请检查 '{param}' 字段的值是否符合要求。" | "Invalid request parameters. Please check that the value of '{param}' meets the requirements." |
| `context_length_exceeded` | "输入内容过长，超过了模型的最大上下文限制（{max_tokens} tokens）。请缩减输入文本长度后重试。" | "Input exceeds the model's maximum context length ({max_tokens} tokens). Please reduce the input length and retry." |
| `content_policy_violation` | "输入内容触发了安全策略，请修改后重试。如涉及误判，请联系客服申诉。" | "Input violates content safety policy. Please revise and retry. Contact support if you believe this is a false positive." |

### 4.5 服务端/上游类 (server_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `upstream_error` | "上游服务暂时异常，请稍后重试。如问题持续，请提供 Request ID '{request_id}' 联系支持团队。" | "Upstream service temporarily unavailable. Please retry later. If the issue persists, contact support with Request ID '{request_id}'." |
| `service_overloaded` | "服务当前负载较高，请稍后再试。建议开启流式传输以降低超时概率。" | "Service is currently overloaded. Please try again later. We recommend enabling streaming to reduce timeout probability." |
| `gateway_timeout` | "网关请求上游超时，请稍后重试。如频繁出现，建议缩短请求长度或开启流式输出。" | "Gateway timed out waiting for upstream. Please retry later. If this occurs frequently, consider shortening requests or enabling streaming." |

### 4.6 异步任务类 (task_xxx)

| ErrorCode | 用户提示话术（中文） | 用户提示话术（English） |
|-----------|-------------------|------------------------|
| `task_failed` | "任务执行失败，原因：{reason}。已自动退还额度，请修改参数后重新提交。" | "Task execution failed: {reason}. Quota has been automatically refunded. Please revise parameters and resubmit." |
| `task_timeout` | "任务执行超时（超过 {timeout}），已自动取消并退还额度。建议缩短输入或减少生成数量后重试。" | "Task timed out after {timeout}. It has been automatically cancelled and quota refunded. Consider reducing input length or generation count." |

---

## 五、关键结论

1. **OpenAI 格式是行业事实标准**：new-api 的 OpenAI 兼容转换 (`ToOpenAIError`) 是正确方向，应继续强化。
2. **Claude 的 `request_id` + 嵌套 `error` 结构值得借鉴**：调试体验优于 OpenAI（OpenAI 的 request_id 只在 Header 中）。
3. **LiteLLM 的 `provider_specific_fields` 是网关的差异化价值**：作为聚合 40+ 上游的网关，保留上游原始错误信息对排查至关重要。
4. **new-api 的架构基础已很扎实**：`NewAPIError` + `ErrorCode` + `RelayErrorHandler` + 敏感信息脱敏，均属于生产级设计。
5. **最大提升空间在"用户指导动作"**：当前错误消息以透传上游为主，缺少结构化的 "What to do" 提示。建立 `ErrorCode → 话术模板 → 多语言 → 文档链接` 的映射体系，是本次项目最核心的可落地改进。
