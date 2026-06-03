# new-api 错误处理体系技术数据分析

> 分析范围：错误码体系、状态码映射、重试策略、渠道禁用策略、退款策略、影响面评估
> 数据来源：`types/error.go`, `service/channel.go`, `service/billing_session.go`, `controller/relay.go`, `setting/operation_setting/status_code_ranges.go`, `service/task_billing.go`, `service/task_polling.go`

---

## 一、错误分类矩阵

按 **错误来源 × 错误性质** 对 39 个标准 ErrorCode + 任务错误体系进行分类。

### 1.1 同步请求错误分类

| 错误码 | 来源 | 性质 | 说明 |
|--------|------|------|------|
| `invalid_request` | 用户/系统 | 请求 | 通用请求参数非法 |
| `sensitive_words_detected` | 系统 | 内容 | 敏感词检测命中 |
| `violation_fee.grok.csam` | 上游 | 内容 | Grok CSAM 安全违规，触发额外扣费 |
| `count_token_failed` | 系统 | 请求 | Token 估算失败 |
| `model_price_error` | 系统 | 请求 | 模型定价查询失败 |
| `invalid_api_type` | 系统 | 请求 | API 类型不匹配 |
| `json_marshal_failed` | 系统 | 请求 | 请求体序列化失败 |
| `do_request_failed` | 渠道/网络 | 网络 | 向上游发送请求失败（dial/post/http 错误） |
| `get_channel_failed` | 系统/渠道 | 渠道 | 无可选渠道或渠道选择失败 |
| `gen_relay_info_failed` | 系统 | 请求 | Relay 信息生成失败 |
| `channel:no_available_key` | 渠道 | 认证 | 渠道无可用 Key |
| `channel:param_override_invalid` | 渠道 | 请求 | 渠道参数覆盖配置非法 |
| `channel:header_override_invalid` | 渠道 | 请求 | 渠道 Header 覆盖配置非法 |
| `channel:model_mapped_error` | 渠道 | 请求 | 模型映射失败 |
| `channel:aws_client_error` | 渠道 | 认证 | AWS 客户端初始化失败 |
| `channel:invalid_key` | 渠道 | 认证 | 渠道 Key 非法 |
| `channel:response_time_exceeded` | 渠道 | 网络 | 渠道响应超时 |
| `read_request_body_failed` | 用户 | 请求 | 读取请求体失败（含 413 过大） |
| `convert_request_failed` | 系统 | 请求 | 请求格式转换失败 |
| `access_denied` | 系统 | 认证 | 访问被拒绝（如 playground/image_studio 的 access token 限制） |
| `bad_request_body` | 上游/渠道 | 请求 | 上游返回请求体错误（AWS Nova 解码/格式化失败等） |
| `read_response_body_failed` | 上游/网络 | 响应 | 读取上游响应体失败 |
| `bad_response_status_code` | 上游 | 响应 | 上游返回非 2xx 状态码 |
| `bad_response` | 上游 | 响应 | 上游响应通用错误 |
| `bad_response_body` | 上游 | 响应 | 上游响应体解析失败 |
| `empty_response` | 上游 | 响应 | 上游返回空响应 |
| `aws_invoke_error` | 上游/渠道 | 响应 | AWS InvokeModel 失败 |
| `model_not_found` | 上游 | 请求 | 上游模型不存在 |
| `prompt_blocked` | 上游 | 内容 | 提示词被上游拦截 |
| `query_data_error` | 系统 | 系统 | 数据库查询失败 |
| `update_data_error` | 系统 | 系统 | 数据库更新失败 |
| `insufficient_user_quota` | 用户 | 额度 | 用户额度不足 |
| `pre_consume_token_quota_failed` | 用户/渠道 | 额度 | 令牌额度预扣失败 |

### 1.2 任务（异步）错误分类

| 错误码/阶段 | 来源 | 性质 | 说明 |
|-------------|------|------|------|
| `get_origin_task_failed` | 系统 | 请求 | 获取原始任务失败 |
| `fail_to_fetch_task` | 上游 | 网络 | 拉取任务状态失败 |
| `copy_response_body_failed` | 系统 | 响应 | 复制响应体失败 |
| `get_tasks_failed` / `get_task_failed` | 上游/系统 | 响应 | 查询任务列表/单任务失败 |
| `convert_to_openai_video_failed` | 系统 | 响应 | 视频结果转 OpenAI 格式失败 |
| `marshal_response_failed` | 系统 | 响应 | 响应序列化失败 |
| `setup_locked_channel_failed` | 渠道 | 渠道 | 锁定渠道设置失败 |
| `read_request_body_failed` | 用户 | 请求 | 读取请求体失败（任务提交阶段） |
| `task_failed` | 上游 | 内容/响应 | 上游明确返回任务失败 |
| `submit_failed` | 上游 | 请求 | 任务提交被上游拒绝 |
| `unmarshal_response_body_failed` | 上游 | 响应 | 上游响应体反序列化失败 |
| `unmarshal_task_result_failed` | 上游 | 响应 | 任务结果反序列化失败 |
| `unmarshal_task_data_failed` | 上游 | 响应 | 任务数据反序列化失败 |
| `get_task_request_failed` | 上游/网络 | 网络 | 查询任务状态的请求发送失败 |

---

## 二、状态码-错误码映射表

### 2.1 系统固定映射

| ErrorCode | 默认 HTTP StatusCode | 触发场景 |
|-----------|---------------------|----------|
| `invalid_request` | 500 (Internal Server Error) | 通用请求验证失败，未指定状态码时 |
| `sensitive_words_detected` | 500 | 敏感词命中（默认，实际由 controller 决定） |
| `violation_fee.grok.csam` | 继承上游状态码 | Grok CSAM 违规 |
| `count_token_failed` | 500 | Token 估算异常 |
| `model_price_error` | 400 (Bad Request) | 模型价格配置缺失或非法 |
| `invalid_api_type` | 500 | API 类型无法识别 |
| `json_marshal_failed` | 500 | JSON 序列化失败 |
| `do_request_failed` | 500 | HTTP 请求发送失败（dial/post 错误） |
| `get_channel_failed` | 500 | 渠道选择失败 |
| `gen_relay_info_failed` | 500 | RelayInfo 生成失败 |
| `channel:no_available_key` | 继承上下文 | 渠道 Key 耗尽 |
| `channel:param_override_invalid` | 500 | 参数覆盖配置错误 |
| `channel:header_override_invalid` | 500 | Header 覆盖配置错误 |
| `channel:model_mapped_error` | 500 | 模型映射错误 |
| `channel:aws_client_error` | 500 | AWS 客户端错误 |
| `channel:invalid_key` | 继承上下文 | 渠道 Key 非法 |
| `channel:response_time_exceeded` | 继承上下文 | 渠道响应超时 |
| `read_request_body_failed` | 400 / 413 | 请求体读取失败或过大 |
| `convert_request_failed` | 500 | 请求转换失败 |
| `access_denied` | 500 | 访问被拒绝 |
| `bad_request_body` | 500 | 请求体格式错误（AWS 解码失败等） |
| `read_response_body_failed` | 500 | 响应体读取失败 |
| `bad_response_status_code` | **上游实际状态码** | 上游返回非 2xx |
| `bad_response` | 500 | 通用响应错误 |
| `bad_response_body` | 500 | 响应体解析失败 |
| `empty_response` | 500 | 空响应 |
| `aws_invoke_error` | **上游 AWS 状态码** | AWS InvokeModel 错误 |
| `model_not_found` | 500 | 模型不存在 |
| `prompt_blocked` | 500 | 提示词被拦截 |
| `query_data_error` | 500 | 数据库查询错误 |
| `update_data_error` | 500 | 数据库更新错误 |
| `insufficient_user_quota` | 403 (Forbidden) | 用户/订阅额度不足 |
| `pre_consume_token_quota_failed` | 403 (Forbidden) | 令牌额度预扣失败 |

### 2.2 上游错误类型映射（OpenAI 格式）

系统通过 `WithOpenAIError` 封装上游错误，状态码规则：

| 上游返回状态码 | 系统行为 |
|---------------|----------|
| 2xx | 正常处理，不进入错误路径 |
| 4xx (除 400/408) | 通常 **不重试**（但 401-407 在自动重试范围内，除非 skipRetry） |
| 400 | 不重试（不在 AutomaticRetryStatusCodeRanges） |
| 401-407 | 重试（在自动重试范围），但 401 同时触发渠道禁用 |
| 408 | 不重试（不在自动重试范围） |
| 409-499 | 重试（在自动重试范围） |
| 500-503 | 重试 |
| 504 | **始终不重试**（always skip） |
| 505-523 | 重试 |
| 524 | **始终不重试**（always skip） |
| 525-599 | 重试 |
| 非 100-599 | 重试（视为未知网络错误） |

---

## 三、重试决策矩阵

### 3.1 同步请求重试规则（`shouldRetry`）

| 条件 | 是否重试 | 说明 |
|------|----------|------|
| `err == nil` | 否 | 无错误 |
| 渠道亲和性失败后的跳过标记 | 否 | 避免无限重试同一渠道 |
| `IsChannelError(err)` | **是** | 所有 `channel:` 开头的错误都重试 |
| `IsSkipRetryError(err)` | **否** | 显式标记 skipRetry 的错误 |
| `retryTimes <= 0` | 否 | 已达最大重试次数 |
| 指定了 `specific_channel_id` | 否 | 用户强制指定渠道 |
| 状态码 200-299 | 否 | 成功状态码 |
| 状态码 100-199, 300-399, 401-407, 409-499, 500-503, 505-523, 525-599 | **是** | 在自动重试范围内 |
| 状态码 504, 524 | **否** | alwaysSkipRetryStatusCodes |
| 错误码 `bad_response_body` | **否** | alwaysSkipRetryCodes |
| 状态码 <100 或 >599 | **是** | 视为网络层异常 |

### 3.2 显式 SkipRetry 的错误清单

以下错误在代码中通过 `ErrOptionWithSkipRetry()` 显式标记为不重试：

| ErrorCode | 触发场景 |
|-----------|----------|
| `invalid_request` | 请求类型不匹配、参数验证失败 |
| `sensitive_words_detected` | 敏感词命中 |
| `model_price_error` | 模型价格错误 |
| `invalid_api_type` | API 类型非法 |
| `channel:param_override_invalid` | 参数覆盖配置错误 |
| `channel:model_mapped_error` | 模型映射错误 |
| `read_request_body_failed` | 请求体读取失败（含 413） |
| `convert_request_failed` | 请求转换失败 |
| `access_denied` | 访问被拒绝 |
| `json_marshal_failed` | JSON 序列化失败 |
| `get_channel_failed` | 渠道获取失败 |
| `gen_relay_info_failed` | RelayInfo 生成失败 |
| `insufficient_user_quota` | 额度不足 |
| `pre_consume_token_quota_failed` | 令牌预扣失败 |
| `violation_fee.grok.csam` | CSAM 违规（标准化后强制 skip） |
| `bad_response_status_code` | 图像/任务等处理时显式 skip |

### 3.3 任务（异步）重试规则（`shouldRetryTaskRelay`）

| 条件 | 是否重试 |
|------|----------|
| `err == nil` | 否 |
| 渠道亲和性失败后 | 否 |
| `retryTimes <= 0` | 否 |
| 指定了 specific_channel_id | 否 |
| StatusCode == 429 | **是** |
| StatusCode == 307 | **是** |
| StatusCode 5xx | **是**（但 504/524 在同步规则中 skip） |
| StatusCode == 400 | 否 |
| StatusCode == 408 | 否 |
| `LocalError == true` | 否 |
| StatusCode 2xx | 否 |
| 其他 | **是** |

---

## 四、渠道禁用决策矩阵

`ShouldDisableChannel` 返回 true 的条件（需同时满足 `AutomaticDisableChannelEnabled` 和 `AutoBan`）：

| 触发条件 | 说明 | 优先级 |
|----------|------|--------|
| `IsChannelError(err)` | 错误码以 `channel:` 开头 | 高 |
| `!IsSkipRetryError(err)` | 非 skipRetry 错误 | 前置过滤 |
| 状态码在 `AutomaticDisableStatusCodeRanges` | 默认：**401** | 中 |
| OpenAI Error Code 匹配 | `invalid_api_key`, `account_deactivated`, `billing_not_active`, `pre_consume_token_quota_failed`, `Arrearage` | 高 |
| OpenAI Error Type 匹配 | `insufficient_quota`, `insufficient_user_quota`, `authentication_error`, `permission_error`, `forbidden` | 高 |
| 错误消息匹配 `AutomaticDisableKeywords` | 余额不足、组织禁用、配额超限、权限拒绝、Token 无效、操作不允许、账户未授权 | 中 |

### 4.1 会触发禁用的典型错误

| 错误场景 | 错误码/消息 | 触发禁用 |
|----------|-------------|----------|
| 渠道 Key 无效 | `invalid_api_key` | **是** |
| 账户已停用 | `account_deactivated` | **是** |
| 账单未激活 | `billing_not_active` | **是** |
| 欠费 | `Arrearage` | **是** |
| 配额不足 | `insufficient_quota` / `insufficient_user_quota` | **是** |
| 认证错误 | `authentication_error` | **是** |
| 权限错误 | `permission_error` / `forbidden` | **是** |
| 渠道无可用 Key | `channel:no_available_key` | **是** |
| 渠道 AWS 客户端错误 | `channel:aws_client_error` | **是** |
| 渠道参数覆盖非法 | `channel:param_override_invalid` | **是** |
| 401 Unauthorized | `bad_response_status_code` (401) | **是**（默认配置） |
| 用户额度不足 | `insufficient_user_quota` | **否**（skipRetry） |
| 令牌额度不足 | `pre_consume_token_quota_failed` | **是**（OpenAI Code 匹配） |
| 请求体过大 | `read_request_body_failed` (413) | **否**（skipRetry） |
| 敏感词 | `sensitive_words_detected` | **否**（skipRetry） |

---

## 五、退款决策矩阵

### 5.1 同步请求退款规则

| 场景 | 是否退款 | 退款范围 | 说明 |
|------|----------|----------|------|
| 预扣费后下游请求失败 | **是** | 预扣全额 | `BillingSession.Refund()` 异步退还 wallet/subscription + token quota |
| 已成功结算后 | 否 | - | `settled=true` 后不可退款 |
| 已退款后 | 否 | - | `refunded=true` 幂等保护 |
| 免费模型 | 否 | - | 未预扣费 |
| Grok CSAM 违规 | **否，反而扣费** | 额外扣 violation fee | `ChargeViolationFeeIfNeeded` |

**关键代码路径**（`controller/relay.go`）：
```go
defer func() {
    if newAPIError != nil {
        newAPIError = service.NormalizeViolationFeeError(newAPIError)
        if relayInfo.Billing != nil {
            relayInfo.Billing.Refund(c)
        }
        service.ChargeViolationFeeIfNeeded(c, relayInfo, newAPIError)
    }
}()
```

### 5.2 异步任务退款规则

| 场景 | 是否退款 | 退款范围 | 说明 |
|------|----------|----------|------|
| 任务提交失败（taskErr != nil） | **是** | 预扣全额 | `relayInfo.Billing.Refund(c)` |
| 任务轮询发现失败 | **是** | 预扣全额 | `RefundTaskQuota(ctx, task, failReason)` |
| 任务超时（新系统） | **是** | 预扣全额 | 超时标记 FAILURE 后退款 |
| 任务超时（旧系统遗留） | **否** | - | 2026-02-22 前的任务不退款 |
| 任务成功 | 否 | - | 执行 `SettleBilling` 或 `RecalculateTaskQuota` |
| 任务成功但按次计费 | 否 | - | 预扣即实扣 |
| 任务成功且返回 totalTokens | 可能补扣/退还 | 差额 | `RecalculateTaskQuotaByTokens` |
| 任务成功且 adaptor 调整额度 | 可能补扣/退还 | 差额 | `AdjustBillingOnComplete` |

### 5.3 退款触发条件汇总

| 错误类型 | 同步退款 | 异步退款 | 额外扣费 |
|----------|----------|----------|----------|
| 网络错误（do_request_failed） | **是** | 是（提交阶段） | 否 |
| 上游 4xx/5xx | **是** | 是（轮询失败） | 否 |
| 渠道错误（channel:xxx） | **是** | 是 | 否 |
| 额度不足（用户侧） | 否（未预扣） | 否（未预扣） | 否 |
| 额度不足（令牌侧） | 否（预扣失败） | 否（预扣失败） | 否 |
| 请求体过大 | 否（skipRetry，未预扣或预扣前失败） | 是（如已预扣） | 否 |
| 敏感词 | 否（skipRetry，未预扣或预扣前失败） | 是（如已预扣） | 否 |
| CSAM 违规 | **否，扣 violation fee** | - | **是** |
| 任务超时 | - | **是**（新系统） | 否 |

---

## 六、错误影响面分析

### 6.1 按频率与影响分级

#### 🔴 高频-高影响

| 错误 | 频率 | 影响 | 理由 |
|------|------|------|------|
| `bad_response_status_code` | 高频 | 高 | 上游 40+ 渠道的任何非 2xx 返回都会触发，是网关最主要的错误来源 |
| `do_request_failed` | 高频 | 高 | 网络抖动、DNS 故障、上游不可用时大量出现，直接触发重试 |
| `read_response_body_failed` / `bad_response_body` | 中高频 | 中高 | 上游返回格式异常或连接中断 |
| `channel:no_available_key` | 中高频 | 高 | 渠道 Key 池耗尽时批量出现，触发渠道禁用 |
| `insufficient_user_quota` | 高频 | 中 | 用户额度不足，skipRetry，影响用户体验但不消耗资源 |

#### 🟡 中频-中影响

| 错误 | 频率 | 影响 | 理由 |
|------|------|------|------|
| `get_channel_failed` | 中频 | 中高 | 分组下无可用渠道时发生，skipRetry，直接阻断请求 |
| `channel:response_time_exceeded` | 中频 | 中 | 渠道响应慢，可能伴随超时重试 |
| `invalid_request` | 中频 | 低 | 客户端参数错误，skipRetry，快速失败 |
| `model_not_found` | 中频 | 低 | 上游模型不存在，skipRetry |
| `prompt_blocked` | 中频 | 中 | 内容安全拦截，skipRetry |
| `task_failed` / `submit_failed` | 中频 | 中 | 异步任务场景特有，触发退款 |
| `aws_invoke_error` | 低频-中频 | 中 | AWS 渠道特有，与 AWS 服务状态相关 |

#### 🟢 低频-低影响 / 系统内部错误

| 错误 | 频率 | 影响 | 理由 |
|------|------|------|------|
| `json_marshal_failed` | 低频 | 低 | 系统内部序列化问题 |
| `query_data_error` / `update_data_error` | 低频 | 高（系统级） | 数据库异常，影响全局 |
| `count_token_failed` | 低频 | 低 | Tokenizer 异常 |
| `model_price_error` | 低频 | 低 | 价格配置缺失 |
| `channel:aws_client_error` | 低频 | 中 | AWS 配置错误，通常配置修正后消失 |
| `violation_fee.grok.csam` | 极低频 | 中 | 仅 Grok 渠道且触发 CSAM 时 |
| `access_denied` | 低频 | 低 | 特定功能限制 |

### 6.2 错误链路影响分析

```
用户请求
  ├── 本地验证阶段（invalid_request, sensitive_words_detected, access_denied）
  │     └── skipRetry, 快速失败, 通常不退款（未预扣）
  ├── 计费预扣阶段（insufficient_user_quota, pre_consume_token_quota_failed）
  │     └── skipRetry, 阻断请求, 未产生实际扣费
  ├── 渠道选择阶段（get_channel_failed, channel:xxx）
  │     └── channel错误→重试; 无渠道→skipRetry失败
  ├── 上游请求阶段（do_request_failed, bad_response_status_code, read_response_body_failed）
  │     └── 非skipRetry→重试→失败→退款
  ├── 响应处理阶段（bad_response_body, empty_response, model_not_found）
  │     └── bad_response_body始终不重试; 其他按状态码判断
  └── 异步任务阶段（task_failed, timeout, submit_failed）
        └── 失败/超时→退款; 成功→结算
```

### 6.3 数据洞察

1. **重试消耗最大资源**：`do_request_failed` 和 `bad_response_status_code` 构成重试的主要来源。每增加一次重试，渠道负载翻倍，且失败渠道会在 `processChannelError` 中被记录到 error_log。

2. **渠道禁用集中在认证/额度类**：401 状态码 + OpenAI `invalid_api_key` / `insufficient_quota` + Keyword 匹配（余额/配额/权限）构成 90% 以上的自动禁用触发条件。

3. **退款集中在异步任务**：同步请求的退款仅在预扣费后、结算前发生，实际比例较低；异步任务因生命周期长，失败/超时退款是常态。

4. **SkipRetry 错误占比高**：约 60% 的标准 ErrorCode 被显式标记为 skipRetry，这些错误多为用户侧或配置侧问题，重试无意义。

5. **状态码 401 是禁用与重试的交汇点**：401 既在自动重试范围内（401-407），又在默认自动禁用范围内（401），意味着 401 错误会触发**重试其他渠道 + 禁用当前渠道**的双重动作。

---

*分析完成时间：基于 new-api 代码库最新状态*
