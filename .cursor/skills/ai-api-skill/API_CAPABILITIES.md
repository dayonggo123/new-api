# 万象Ai API 能力文档

> 自动生成于 API 接口，平台能力更新时请重新拉取

- **平台**: 万象Ai
- **版本**: 2026-05-14
- **基础地址**: https://api.lk888.ai/api
- **认证**: Bearer {api_key}

## 接口列表

### 模型查询

#### 获取模型列表
- **方法**: `GET`
- **路径**: `/v1/skills/models`
- **说明**: 按类型查询平台所有可用模型。返回每个模型的名称、展示名称、类型、功能标签和简介。
- 不传 type 参数返回所有类型的模型
- type=chat 时只返回 gpt/o1/o3/chatgpt/claude/gemini 前缀的语言模型，并额外返回 api_format（调用格式：openai/anthropic/gemini）和 api_endpoint（对应的请求路径）
- type=image/video/audio/tts/music 返回对应类型的媒体模型

响应字段说明：
- name: 模型标识名，调用接口时传此值
- display_name: 展示用的中文名称
- type: 模型类型（chat/image/video/audio/tts/music）
- tags: 功能标签数组，如["文生视频","图生视频"]
- description: 模型简介
- input_hint: 输入提示文案
- api_format: [仅chat] 调用格式，openai/anthropic/gemini
- api_endpoint: [仅chat] 对应请求路径，如 /v1/chat/completions
- **参数**:
  - `type` (string, 可选): 按模型类型筛选，不传则返回全部
- **提示**: 1. chat 类型只返回主流语言模型（gpt/o1/o3/chatgpt/claude/gemini 前缀），其他特殊模型不在此列表中。
2. 媒体模型（image/video/audio）返回全部可用模型。TTS 语音合成和 music 音乐模型统一归类为 audio 类型，使用 type=audio 可查询到。
3. 每个 chat 模型的 api_format 告诉你该用哪种格式调用：openai 用 /v1/chat/completions，anthropic 用 /v1/messages，gemini 用 /v1beta/models/{model}:{action}。
4. 要获取模型详细参数用 /v1/skills/models/{name}，要获取价格用 /v1/skills/models/{name}/pricing。
5. aliases 字段：可选。当模型 display_name 是缩写但 description 含业内通用品牌名（如 Seedance / Veo / Hailuo / Kling 等）时，系统会自动把品牌名提到 aliases 数组里，方便按品牌名检索。display_name 已含的品牌名不会重复列出。例如按 "seedance" 检索可命中 display_name="SD 2.0 首尾帧" 但 aliases 包含 "Seedance" 的模型。

#### 获取模型功能与参数
- **方法**: `GET`
- **路径**: `/v1/skills/models/{model_name}`
- **说明**: 查询单个模型的功能信息和参数列表，不含价格。

响应字段说明：
- name/display_name/type/tags/description: 同模型列表接口
- input_hint: 提示用户输入什么，如"描述视频内容"
- params: 参数定义数组，每个参数包含：
  - name: 参数标识名，调用时传入 params 对象的 key
  - label: 参数中文名称
  - type: 参数类型，select=下拉选择，textarea=文本输入，number=数字输入，upload=文件上传，switch=开关
  - required: 是否必填
  - default: 默认值
  - options: [仅select类型] 可选项数组，每项含 label(显示名)/value(传入值)/is_default
  - description: 参数说明

调用媒体生成接口时，将此处获取的参数放入请求体的 params 对象中。
- **参数**:
  - `model_name` (string, 必填): 模型名称，如 grok-video-3
- **提示**: 1. 此接口只返回功能和参数，不含价格。查价格请用 /v1/skills/models/{name}/pricing。
2. params 数组定义了调用媒体生成接口时可传的参数。type=select 的参数必须从 options 中选取 value 值，不能自拟。
3. type=upload 的参数表示需要传入文件（图片/视频/音频）的可公开访问 URL。单个文件可传字符串 "https://example.com/image.jpg"，多个文件传数组 ["https://example.com/1.jpg","https://example.com/2.jpg"]，两种格式均支持。参数描述中会标注支持的文件数量范围。平台不提供文件上传/托管服务，请自行将文件上传至对象存储服务后传入 URL。
4. 语言模型（chat类型）没有 params，调用方式参见对应的 Chat API 接口文档。
5. aliases 字段：可选，同 list_models 接口。当 display_name 是缩写但 description 含业内通用品牌名（如 Seedance / Veo / Hailuo / Kling 等）时，系统会自动把品牌名提到 aliases 数组里，可以按品牌名检索。display_name 已含的品牌名不会重复列出。

#### 获取模型完整价格
- **方法**: `GET`
- **路径**: `/v1/skills/models/{model_name}/pricing`
- **说明**: 查询模型所有渠道分组的完整价格信息，包括参数价格变动。默认返回全量渠道分组，每个分组含 is_active 字段标识当前是否启用。注意：is_active=false 的分组并非永久关闭，平台会根据供应商状态随时启用或关闭渠道分组，因此展示价格时应包含所有分组供用户参考。传 ?status=active 可仅获取当前正在运行的分组。
- **参数**:
  - `model_name` (string, 必填): 模型名称
  - `status` (string, 可选): 筛选条件。不传或为空返回全部分组；传 active 仅返回当前启用的分组
- **提示**: 1. 默认返回所有渠道分组，每个分组有 is_active 字段。is_active=false 表示该分组当前暂停服务，但随时可能重新启用，不代表永久下线。
2. 建议向终端用户展示全部分组价格（含暂停的），因为这些分组可能随时恢复。
3. 传 ?status=active 仅返回当前正在运行的分组，适用于只关心实时可用渠道的场景。
4. 只返回分组名称，不暴露上游供应商信息。价格已含代理商加成。
5. 若某参数选项未出现在分组的 option_prices 中，表示该选项使用分组的基础价格（base_price），无额外加价。

### 调用说明

#### 通用调用说明
- **方法**: `GET`
- **路径**: `/v1/skills/guide`
- **说明**: 返回平台所有模型的通用调用指南，包含以下内容：

1. 语言模型三种调用格式：
   - OpenAI 格式：POST /v1/chat/completions，适用于 gpt/o1/o3/chatgpt 前缀模型
   - Anthropic 格式：POST /v1/messages，适用于 claude 前缀模型
   - Gemini 格式：POST /v1beta/models/{model}:{action}，适用于 gemini 前缀模型
   每种格式含请求示例和响应示例

2. 媒体模型异步轮询流程：
   - 第一步：POST /v1/media/generate 提交任务，获取 task_id
   - 第二步：GET /v1/skills/task-status?task_id=xxx 轮询状态
   - 轮询间隔建议5秒，is_final=true 时停止

3. 价格计算公式：
   - 按次计费：最终价格 = 基础价格 × 参数系数 + 参数加价
   - 按token计费：费用 = 输入token数 × 输入单价 + 输出token数 × 输出单价

4. 渠道策略说明：
   - 价格优先：自动选择最便宜的可用渠道
   - 速度优先：自动选择响应最快的可用渠道
   - 成功率优先：自动选择成功率最高的可用渠道
   策略在用户的 API Key 设置中配置，调用时无需指定
- **提示**: 1. 此接口返回的是通用调用指南，所有模型共用，不是某个具体模型的说明。
2. 语言模型支持流式输出（stream:true），媒体模型只支持异步轮询，不支持流式。
3. 渠道策略由用户在平台网站的 API Key 设置中配置，调用接口时无需且不能指定渠道或策略。
4. 所有接口均需在 Header 中携带 Authorization: Bearer {API_KEY} 进行认证。

### 任务管理

#### 查询任务状态
- **方法**: `GET`
- **路径**: `/v1/skills/task-status`
- **说明**: 查询媒体生成任务的实时状态。提交生成任务后，通过此接口轮询任务进度和结果。

响应字段说明：
- task_id: 任务ID
- model: 使用的模型名称
- status: 任务状态文本，如"排队中""生成中""生成完成""生成失败"
- status_group: 状态分组，"等待中"/"处理中"/"已完成"/"失败"
- progress: 进度百分比，如"0%"、50%"、"100%"
- is_final: 是否为终态。true 表示任务已结束（成功或失败），必须停止轮询
- result_url: 生成结果的下载地址，仅成功时有值
- result_type: 结果类型，video/image/audio 等
- cost: 实际扣费的算力值
- channel_group: 实际使用的渠道分组名称
- error: 失败时的错误信息
- created_at: 任务创建时间
- completed_at: 任务完成时间，未完成时为空
- duration_seconds: 从创建到完成的耗时（秒）
- **参数**:
  - `task_id` (integer, 必填): 任务ID
- **提示**: 1. 【重要】判断任务是否完成只看 state 和 is_final，不要看 progress 数值。很多异步模型（如 nano-banana-2 / DM 通道、多数视频模型）上游不返回中间 progress，progress 会全程保持 "0"，仅在完成瞬间跳 "100"——这不是卡住，不要提前重试。
2. 当 is_final=true 时必须停止轮询，不要继续请求。
3. 推荐轮询节奏：提交后先等 5-10 秒再开始首次轮询（过早轮询会看到 state=pending、progress=0 是正常现象），之后每 5 秒轮询一次，不要太频繁。
4. 如果轮询超过 7200 秒（7200秒=2小时）任务仍未完成，可认为超时，停止轮询并提示用户。
5. cost 字段语义：代表本次任务的实际成本（单位：算力）。成功任务=运行中按实扣金额；失败任务平台会自动全额退款，此时 cost=0 且 refunded=true；pending 未运行时 cost=0。不需要再关联消费记录表才能知道是否已退款。
6. refunded 字段：false=未退款或未发生退款（含成功/运行中/异常仅扣费未退款场景）；true=任务失败且平台已自动退回全额。可用来向用户明确交代“这笔任务失败了但钱已退”。
7. refunded_amount 字段：refunded=true 时为实际退还金额（算力）；其他场景为 0。
8. channel_group 返回实际使用的渠道分组名称，可用于账单展示和成本记录。
9. 此接口是 /v1/media/status 的增强版，额外返回 model、created_at、completed_at、duration_seconds、channel_group、refunded、refunded_amount 字段。
10. error 字段：无错误时为 null，有错误时为字符串。判断任务是否失败请检查 state=="failed"（或 status_group=="失败"），不要只靠 error != null。
11. input_files 字段：始终为数组类型，无输入文件时为空数组 []。【名称重叠提醒】input_files 只是本响应的统一字段名，调 /v1/media/generate 提交任务时，上传参数名不是 input_files，而是模型详情里实际的 upload 参数名（如 images / image_url / videos / attachments 等），请通过 /v1/skills/models/{name} 查看。
12. 【重要】duration_seconds 是【任务处理总耗时】（completed_at - created_at 的墙钟秒数，含排队/上游生成/下载落库），不是输出视频/音频本身的时长。输出媒体本身的时长由提交时 params.duration 决定（需查对应任务的提交参数），本接口不返回媒体本身时长。

### 账户信息

#### 查询算力余额
- **方法**: `GET`
- **路径**: `/v1/skills/balance`
- **说明**: 查询当前 API Key 对应用户的算力余额和 Key 额度使用情况。

响应字段说明：
- balance: 用户账户的算力余额（注意：单位是算力，不是人民币）
- unit: 余额单位，固定为"算力"
- api_key_quota: API Key 的额度信息
  - used: 该 Key 已使用的算力
  - limit: 该 Key 的总额度上限，0 表示不限额
  - remaining: 该 Key 剩余可用额度，仅在 limit>0 时返回
- **提示**: 1. 余额单位是算力，不是人民币。展示时用"算力"而不是"元"。
2. api_key_quota.limit=0 表示该 Key 不限额，此时不会返回 remaining 字段。
3. 余额不足时应提示用户前往平台官网充值算力。
4. 建议在调用付费接口前先查询余额，避免因余额不足导致任务失败。

#### 查询消费明细
- **方法**: `GET`
- **路径**: `/v1/skills/usage`
- **说明**: 查询最近 N 天本 API Key（或跨 Key 按账户）的算力消费。默认返回按模型聚合的汇总（调用次数/成功数/失败数/实际扣费/退款金额）；detail=1 返回按任务倒序的最近消费记录。失败但已退款的任务 cost 在响应里会处理为 0，refunded_amount 会单独给出，与 /v1/skills/task-status 一致。
- **参数**:
  - `scope` (string, 可选): 范围：key 仅本次调用使用的 API Key（默认）；user 同账户下所有 API Key 汇总。
  - `days` (integer, 可选): 以当前时间为终点向前推 N 天（默认 1，最大 30）；start_time 和 end_time 同时传时本参数被忽略。
  - `start_time` (string, 可选): 起始时间。支持 RFC3339、YYYY-MM-DD HH:MM:SS、YYYY-MM-DD；与 end_time 必须同时传。
  - `end_time` (string, 可选): 结束时间。格式同 start_time；范围不得超过 30 天。
  - `model` (string, 可选): 过滤到某个模型（传模型名称，取 /v1/skills/models 中的 name 字段）。
  - `detail` (string, 可选): detail=1 返回 records 记录列表（含 task_id），不传或 0 返回按模型聚合的 by_model 数组。
  - `limit` (integer, 可选): 仅 detail=1 生效；默认 50，最大 200。
  - `offset` (integer, 可选): 仅 detail=1 生效；从 0 起。
- **提示**: 1) 默认 scope=key 只看本次调用使用的 API Key；scope=user 跨 Key 汇总。
2) 默认 days=1（近 24h），最多 30 天；如需更精确使用 start_time/end_time（同时传），两者冲突时以 start_time/end_time 为准。
3) 时间格式支持 RFC3339 / YYYY-MM-DD HH:MM:SS / YYYY-MM-DD；不传时区默认服务器本地时区。
4) ?model=xxx 可过滤到某个模型。
5) detail=1 时返回的财场变为 records 数组，每条含：task_id, model, model_type, state(success/failed/running/pending), cost, refunded, refunded_amount, created_at, completed_at；顶层额外带 total/limit/offset 翻页字段。limit 默认 50，最大 200；offset 从 0 起。
6) 该接口不读实时并发资源，只提供历史消费汇总；实时余额调 /v1/skills/balance，实时任务状态调 /v1/skills/task-status。
7) 按任务表聚合，后台以提交时间为范围过滤；正在排队/生成中的任务也计入，成功/失败后 cost 会随实际结算金额变动。
8) 失败但已退款的任务，cost 在详情响应里会被处理为 0，refunded=true 且 refunded_amount 为实际退款算力；汇总中则在 refunded_count/refunded_cost 中汇总，by_model[i].cost 为净额（已减退款）。

### 反馈

#### 意见反馈与BUG提交
- **方法**: `POST`
- **路径**: `/v1/skills/feedback`
- **说明**: AI 在使用平台接口过程中，遇到以下情况应主动提交反馈：1) 接口调用报错或返回异常，提交错误信息和调用参数便于平台定位修复；2) 文档说明不清晰或与实际行为不一致，帮助平台完善文档；3) 功能建议或缺少的能力，供平台开发团队参考。这是平台与AI之间的沟通桥梁，积极提交有助于平台持续改进。
- **参数**:
  - `type` (string, 必填): 反馈类型
  - `question` (string, 必填): 问题内容
  - `endpoint` (string, 可选): 相关接口路径
  - `context` (string, 可选): 当时操作背景
  - `ai_tool` (string, 可选): AI工具名称
- **提示**: 每个API Key每小时最多10条，超过返回429。遇到报错时应尽量提供完整信息：调用的接口路径、请求参数、错误信息、操作步骤。不要因为“不确定是不是BUG”就不提交，平台开发团队会判断处理。

#### 查询反馈处理结果
- **方法**: `GET`
- **路径**: `/v1/skills/feedback`
- **说明**: 通过反馈ID查询之前提交的反馈的处理状态和结果
- **参数**:
  - `id` (integer, 必填): 反馈ID，提交反馈时返回的 feedback_id
- **提示**: 1. 参数 id 为之前提交反馈时返回的 feedback_id。
2. 只能查询自己提交的反馈，API Key 不匹配会返回 403。
3. status 可能的值：未处理、已处理、已忽略。
4. resolution 字段在状态为“已处理”时包含修复说明。

### 语言模型调用

#### OpenAI Chat Completions
- **方法**: `POST`
- **路径**: `/v1/chat/completions`
- **说明**: 完全兼容 OpenAI Chat Completions API。可直接使用 OpenAI 官方 SDK，只需将 base_url 指向本平台即可。

适用模型：gpt/o1/o3/chatgpt 前缀的所有模型

主要参数：
- model: 模型名称（必填）
- messages: 消息数组，每条含 role(system/user/assistant) 和 content（必填）
- stream: 是否流式输出，true 为 SSE 流式，false 为一次性返回（默认false）
- temperature: 温度参数 0-2（可选）
- max_tokens: 最大输出 token 数（可选）

响应字段：
- choices[0].message.content: AI 回复内容
- usage.prompt_tokens: 输入消耗的 token 数
- usage.completion_tokens: 输出消耗的 token 数
- **提示**: 1. 认证方式：Header 中携带 Authorization: Bearer {API_KEY}。
2. stream=true 时返回 SSE 流，每行格式为 data: {json}，最后一行为 data: [DONE]。
3. 若使用 OpenAI SDK，只需设置 base_url 和 api_key，其他代码与官方完全一致。
4. 报错时返回 {"error": {"message": "错误说明", "type": "错误类型"}}。
5. 401 表示 API Key 无效，402 表示余额不足，429 表示请求太频繁。
6. 本平台不识别请求体里的 channel_group 字段，也不识别请求头里的 X-Channel-Group；分组路由完全由 API Key 的渠道策略（价格优先/速度优先/成功率优先）决定，见 /v1/skills/guide 的 channel_strategy。自带这些字段只会被透传给上游（可能触发上游自己的行为），不会改变本平台的路由结果。若需在同一程序里对同一模型做分组 fallback（例如官方→官转→直连），请为每个目标分组各建一把密钥、各配不同策略，然后在客户端自行切换。

#### OpenAI Responses
- **方法**: `POST`
- **路径**: `/v1/responses`
- **说明**: 兼容 OpenAI 新版 Responses API 格式。相比 Chat Completions 更简洁，input 可直接传字符串。

适用模型：gpt/o1/o3/chatgpt 前缀的所有模型

主要参数：
- model: 模型名称（必填）
- input: 输入内容，可以是字符串或消息数组（必填）
- stream: 是否流式输出（可选）

响应字段：
- output[0].content[0].text: AI 回复内容
- usage.input_tokens: 输入 token 数
- usage.output_tokens: 输出 token 数
- **提示**: 1. 认证方式：Header 中携带 Authorization: Bearer {API_KEY}。
2. input 可以直接传字符串（简单场景）或消息数组（多轮对话）。
3. 与 Chat Completions 用同样的模型，二者选其一即可。
4. 报错格式和状态码与 Chat Completions 一致。
6. 本平台不识别请求体里的 channel_group 字段，也不识别请求头里的 X-Channel-Group；分组路由完全由 API Key 的渠道策略（价格优先/速度优先/成功率优先）决定，见 /v1/skills/guide 的 channel_strategy。自带这些字段只会被透传给上游（可能触发上游自己的行为），不会改变本平台的路由结果。若需在同一程序里对同一模型做分组 fallback（例如官方→官转→直连），请为每个目标分组各建一把密钥、各配不同策略，然后在客户端自行切换。

#### Anthropic Messages
- **方法**: `POST`
- **路径**: `/v1/messages`
- **说明**: 完全兼容 Anthropic Messages API。可直接使用 Anthropic 官方 SDK，只需将 base_url 指向本平台。

适用模型：claude 前缀的所有模型

主要参数：
- model: 模型名称（必填）
- messages: 消息数组，每条含 role(user/assistant) 和 content（必填）
- max_tokens: 最大输出 token 数（必填，Anthropic 格式强制要求）
- system: 系统提示词，单独字段而非放在 messages 中（可选）
- stream: 是否流式输出（可选）

响应字段：
- content[0].text: AI 回复内容
- usage.input_tokens: 输入 token 数
- usage.output_tokens: 输出 token 数
- stop_reason: 停止原因，"end_turn" 表示正常结束
- **提示**: 1. 认证方式：Header 中携带 Authorization: Bearer {API_KEY}（注意：不是 x-api-key，本平台统一使用 Bearer Token）。
2. max_tokens 是必填参数，不传会报错。建议设为 1024 或更高。
3. system 提示词是单独的字段，不要放在 messages 数组中，这是 Anthropic 格式与 OpenAI 的主要区别。
4. 若使用 Anthropic SDK，注意将 base_url 指向本平台，api_key 填写本平台的 API Key。
5. 报错时返回 {"error": {"message": "错误说明", "type": "错误类型"}}。
6. 本平台不识别请求体里的 channel_group 字段，也不识别请求头里的 X-Channel-Group；分组路由完全由 API Key 的渠道策略（价格优先/速度优先/成功率优先）决定，见 /v1/skills/guide 的 channel_strategy。自带这些字段只会被透传给上游（可能触发上游自己的行为），不会改变本平台的路由结果。若需在同一程序里对同一模型做分组 fallback（例如官方→官转→直连），请为每个目标分组各建一把密钥、各配不同策略，然后在客户端自行切换。

#### Gemini Generate Content
- **方法**: `POST`
- **路径**: `/v1beta/models/{model}:{action}`
- **说明**: 完全兼容 Google Gemini API。可直接使用 Google AI SDK，只需将 base_url 指向本平台。

适用模型：gemini 前缀的所有模型

URL 格式：/v1beta/models/{model}:{action}
- {model}: 模型名称，如 gemini-3-pro
- {action}: 操作类型
  - generateContent: 非流式，一次性返回完整结果
  - streamGenerateContent: 流式输出

主要参数：
- contents: 消息数组，每条含 role(user/model) 和 parts（必填）
  - parts 支持的类型：
    - {"text": "文本内容"}: 纯文本
    - {"inlineData": {"mimeType": "类型", "data": "base64编码"}}: 图片/视频/音频/PDF 等文件
- generationConfig: 生成配置，含 temperature/maxOutputTokens 等（可选）

支持的附件类型（通过 inlineData 传入）：
- 图片：image/jpeg, image/png, image/gif, image/webp
- 视频：video/mp4, video/webm, video/mov
- 音频：audio/mp3, audio/wav, audio/ogg, audio/flac
- 文档：application/pdf

响应字段：
- candidates[0].content.parts[0].text: AI 回复内容
- usageMetadata.promptTokenCount: 输入 token 数
- usageMetadata.candidatesTokenCount: 输出 token 数
- **提示**: 1. 认证方式：URL 参数 key={API_KEY} 或 Header 中 Authorization: Bearer {API_KEY}，两种方式均支持。
2. 模型名和操作在 URL 路径中指定，不在请求体中。例如：/v1beta/models/gemini-3-pro:generateContent。
3. Gemini 的角色名称是 user 和 model（不是 assistant）。
4. 若使用 Google AI SDK，将 base_url/api_endpoint 指向本平台，api_key 填写本平台的 API Key。
5. streamGenerateContent 返回的流格式与 Gemini 官方一致。
6. 传入附件（图片/视频/音频/PDF）时，使用 Gemini 原生的 inlineData 格式：在 parts 数组中添加 {"inlineData": {"mimeType": "文件MIME类型", "data": "base64编码内容"}}。与 Gemini 官方 API 格式完全一致，无需额外适配。
7. 附件必须使用 base64 编码内联传入，不支持直接传 URL。若文件在远程服务器，需先下载并转为 base64 后再传入。
8. 本平台不识别请求体里的 channel_group 字段，也不识别请求头里的 X-Channel-Group；分组路由完全由 API Key 的渠道策略决定，见 /v1/skills/guide 的 channel_strategy。若需分组 fallback，请为不同策略各建一把密钥。

### 媒体生成

#### 获取媒体模型列表
- **方法**: `GET`
- **路径**: `/v1/media/models`
- **说明**: 返回所有可用的媒体生成模型及其参数定义。每个模型包含 name、type、label、description 和 params 字段。

params 定义了调用 /v1/media/generate 时可传的参数，包括名称、类型、选项、默认值等。

注意：建议使用 /v1/skills/models 接口替代，信息更完整（含功能标签、展示名称等）。
- **提示**: 1. 此接口是早期版本，建议优先使用 /v1/skills/models 获取模型列表。
2. 两个接口返回的模型数据一致，但 skills 版本额外含 tags、display_name 等字段。

#### 提交媒体生成任务
- **方法**: `POST`
- **路径**: `/v1/media/generate`
- **说明**: 提交图片/视频/音频/TTS/音乐生成任务。提交后返回 task_id，通过轮询接口查询结果。

请求体参数：
- model: 模型名称（必填），从 /v1/skills/models 获取
- prompt: 提示词/文本描述（必填）
- params: 参数对象（可选），从 /v1/skills/models/{name} 获取可用参数

params 用法说明：
- 先调用 /v1/skills/models/{model_name} 获取模型的 params 定义
- 将需要的参数组装为对象，key 是参数的 name，value 是参数值
- 例如模型有 duration 参数（type=select，options含"5"和"10"），则传 {"duration": "5"}
- type=select 的参数必须从 options 中选取 value 值
- type=upload 的参数传入图片/视频的 URL 地址
- 未传的参数使用默认值

响应：
- data.任务id: 任务ID，用于轮询状态
- **参数**:
  - `model` (string, 必填): 模型名称，通过 /v1/skills/models 获取
  - `prompt` (string, 必填): 生成内容的文字描述
  - `params` (object, 可选): 模型专属参数，字段定义来自 /v1/skills/models/{name}。type=upload 的参数传入可公开访问的文件URL
  - `count` (integer, 可选): 生成数量，默认1
- **提示**: 1. 认证方式：Header 中携带 Authorization: Bearer {API_KEY}。
2. 提交后立即返回 task_id（同时为了兼容老版本，仍会返回 任务ids 数组 / 对话组ID 等中文字段），不会等待生成完成。需通过 /v1/skills/task-status?task_id=xxx 轮询结果。
3. 所有模型特定参数必须放在 params 对象内，不要放在请求体顶层。params 中的参数定义来自 /v1/skills/models/{name} 接口，type=select 的参数只能从 options 中选值，不能自拟。
4. type=upload 的参数需要传入可公开访问的文件 URL。平台不提供文件上传/托管服务，请自行将文件上传至 COS、CDN 或其他对象存储服务。upload 参数同时支持“单字符串”和“字符串数组”两种形式（如 "https://x/a.png" 或 ["https://x/a.png","https://x/b.png"]）；禁止传对象数组如 [{"url":"..."}]，会被拒绝。
5. 【参数名 vs 响应字段名 并不相同】请求体里的 upload 参数名由模型决定（常见：images / image_url / img_url / videos / video_url / attachments / reference_urls / audio_url 等），task-status 响应里统一叫 input_files 仅作为取回查看，“input_files” 不是请求参数名。在请求里传 input_files 会被静默忽略。
6. 音乐模型（music-2.5、music-2.5+）在歌曲模式（is_instrumental 不为 instrumental）下，必须在 params 中传入 lyrics（歌词文本），否则会报「歌词不能为空」。
7. TTS 语音合成模型（如 speech-2.8、gemini-2.5-pro-preview-tts）需要先通过 /v1/skills/voices 接口获取可用音色列表，并在 params 中传入对应的 voice_id。speech-2.8 还支持通过 /v1/skills/voices/clone 接口克隆自定义音色。
8. 提交前建议先查询余额（/v1/skills/balance），余额不足会导致任务失败。
9. 同一个 API Key 可同时提交多个任务，并行轮询各自的 task_id 即可。

#### 查询任务状态（原版）
- **方法**: `GET`
- **路径**: `/v1/media/status`
- **说明**: 查询媒体生成任务的实时状态（早期版本）。返回任务进度、结果地址、扣费等信息。

建议使用 /v1/skills/task-status 替代，增强版额外返回：
- model: 模型名称
- created_at: 创建时间
- completed_at: 完成时间
- duration_seconds: 耗时
- channel_group: 渠道分组名称
- **参数**:
  - `task_id` (integer, 必填): 任务ID
- **提示**: 1. 建议优先使用增强版 /v1/skills/task-status，信息更完整。
2. 两个接口的基础字段一致（task_id/status/progress/is_final/result_url/cost等）。

#### 获取可用音色列表
- **方法**: `GET`
- **路径**: `/v1/skills/voices`
- **说明**: 获取当前用户可用的 TTS 音色列表，支持按模型筛选
- **参数**:
  - `model` (string, 可选): Filter by model name: speech-2.8 or gemini-2.5-pro-preview-tts. Omit to get all voices.
- **提示**: 1. 支持 model 参数筛选：?model=speech-2.8 返回用户克隆的音色，?model=gemini-2.5-pro-preview-tts 返回预设音色，不传 model 参数则返回全部。
2. speech-2.8 的音色为用户自己克隆的音色（type=cloned），需通过 /v1/skills/voices/clone 接口创建。
3. gemini-2.5-pro-preview-tts 的音色为平台预设音色（type=preset），无需创建。
4. 克隆音色默认有效期 7 天，首次使用后转为永久。已过期的音色不会在列表中返回。

#### 克隆自定义音色
- **方法**: `POST`
- **路径**: `/v1/skills/voices/clone`
- **说明**: 上传音频文件克隆自定义音色，用于 speech-2.8 模型
- **提示**: 1. 仅支持 speech-2.8 模型的音色克隆。
2. audio_url 必须是可公开访问的音频文件 URL，支持 mp3/wav/flac 等格式，文件不超过 20MB。
3. 克隆会扣除 0.1 算力，余额不足时会返回 402 错误。
4. 音色名称不能重复（同一用户下），不超过 50 个字符。
5. 新克隆的音色有效期 7 天，首次用于生成后自动转为永久。
6. 音频建议 10-60 秒，清晰无杂音的单人语音效果最佳。

## 模型列表

### 语言模型 (chat)

共 22 个模型

- **GPT-5.4** (`gpt-5.4`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.4是OpenAI用于复杂专业工作的前沿模型，具备强大的深度推理、多模态理解和工具调用能力，适用于高难度分析、代码开发与创意写作。

- **Gemini 3.1 Pro** (`gemini-3.1-pro-preview`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索, 哈基米
  - 格式: `gemini`
  - 端点: `/v1beta/models/{model}:{action}`
  - 简介: Gemini 3.1是谷歌迄今为止最智能的模型系列，以先进的推理能力为基础，最适合需要广泛世界知识和跨模态的高级推理的复杂任务。

- **GPT-5.5** (`gpt-5.5`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.5 是 OpenAI 推出的新一代旗舰模型，全面提升对话流畅度、多模态理解与代码能力，适合高质量多轮对话与专业场景。

- **opus-4-7** (`claude-opus-4-7`)
  - 标签: 写代码, 深度思考, 长上下文, 联网搜索
  - 格式: `anthropic`
  - 端点: `/v1/messages`
  - 简介: Claude系列最新一代旗舰模型，在复杂逻辑推理、数学证明及创意写作的细腻度上进一步提升，文采斑斓且极难被检测出是AI。

- **sonnet-4-6** (`claude-sonnet-4-6`)
  - 标签: 写代码, 深度思考, 长上下文, 联网搜索
  - 格式: `anthropic`
  - 端点: `/v1/messages`
  - 简介: Claude Sonnet 4.6 可大规模提供前沿智能，专为编码、代理和企业工作流而打造，均衡型全能选手，以安全拟人著称。

- **GPT-5.5 深度推理** (`gpt-5.5-xhigh`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.5-xhigh 是 OpenAI 推出的最高推理强度版本，专为需要深度逻辑分析、多步骤严谨论证的复杂专业场景设计。

- **Gemini 3 Pro** (`gemini-3-pro-preview`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索, 哈基米
  - 格式: `gemini`
  - 端点: `/v1beta/models/{model}:{action}`
  - 简介: 谷歌最强AI大脑，拥有无限上下文记忆能力，能瞬间吞噬并分析海量文档、整本小说或长视频，回答任意细节问题。

- **GPT-5.5 高推理** (`gpt-5.5-high`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.5-high 是 OpenAI 推出的高强度推理版本，在推理深度与响应速度之间取得良好平衡，适合中高难度推理任务。

- **opus-4-6** (`claude-opus-4-6`)
  - 标签: 写代码, 深度思考, 长上下文, 联网搜索
  - 格式: `anthropic`
  - 端点: `/v1/messages`
  - 简介: Claude系列中最聪明的“逻辑怪兽”，在复杂逻辑推理、数学证明及创意写作的细腻度上排名第一，文采斐然且极难被检测出是AI。

- **GPT-5.4 深度推理** (`gpt-5.4-xhigh`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.4-high是OpenAI推出的高推理强度版本，专为需要深度逻辑分析、复杂专业场景设计的旗舰模型。

- **GPT-5.4 mini** (`gpt-5.4-mini`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.4 mini 将 GPT-5.4 的优势融入到一个更快、更高效的模型中，专为高负载工作而设计。

- **Gemini 3.1 flash** (`gemini-3.1-flash-lite-preview`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索, 极速
  - 格式: `gemini`
  - 端点: `/v1beta/models/{model}:{action}`
  - 简介: Gemini 3.1 系列最具成本效益的多模态模型，可为高频轻量级任务提供最快的性能。最适合处理海量代理任务、简单的数据提取任务，以及预算和速度是主要限制因素的极低延迟应用。

- **Gemini 3 flash** (`gemini-3-flash-preview`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索, 极速, 哈基米
  - 格式: `gemini`
  - 端点: `/v1beta/models/{model}:{action}`
  - 简介: Gemini 3的轻量化版本，主打极速低延迟响应和高性价比，非常适合处理大量数据提取任务或进行即时的语音对话。

- **GPT-5.2 对话** (`gpt-5.2-chat-latest`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索, gpt
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: OpenAI最新旗舰模型，实现了推理与直觉的统一，具备近乎即时的多模态响应速度，是能拆解复杂任务、最像真人的智能助手。

- **claude-4-5** (`claude-haiku-4-5-20251001`)
  - 标签: 深度思考, 写代码, 联网搜索
  - 格式: `anthropic`
  - 端点: `/v1/messages`
  - 简介: 均衡型全能选手，以安全拟人著称，且具备强大的“计算机操作能力”，能模拟人类去点击鼠标、操作软件来帮你完成繁琐工作。

- **GPT-5.4 nano** (`gpt-5.4-nano`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.4 nano 是 GPT-5.4 最轻量、最快速的版本，专为对速度和成本要求极高的任务而设计。

- **GPT-5.2 Codex** (`gpt-5.2`)
  - 标签: 深度思考, 写代码, 联网搜索, gpt
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: 专为代码开发微调的工程级模型，拥有全栈架构师视野，能一次性读取并理解整个项目的代码库，进行系统级架构优化与Bug排查。

- **GPT-5.3 Codex** (`gpt-5.3-codex`)
  - 标签: 写代码, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: 专为代码开发微调的工程级模型，拥有全栈架构师视野，能一次性读取并理解整个项目的代码库，进行系统级架构优化与Bug排查。

- **GPT-5.3 对话** (`gpt-5.3-chat-latest`)
  - 标签: 多轮对话, 多模态, 深度思考, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.3-chat-latest 是 OpenAI 优化的高速对话模型，响应更直接自然，减少说教和拒答，适合日常轻量任务。

- **opus-4-5** (`claude-opus-4-5-20251101`)
  - 标签: 深度思考, 写代码, 联网搜索
  - 格式: `anthropic`
  - 端点: `/v1/messages`
  - 简介: Claude系列中最聪明的“逻辑怪兽”，在复杂逻辑推理、数学证明及创意写作的细腻度上排名第一，文采斐然且极难被检测出是AI。

- **GPT-5.5 低推理** (`gpt-5.5-low`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.5-low 是 OpenAI 推出的轻量推理版本，速度极快且成本最低，适合高并发、轻量型推理任务。

- **GPT-5.5 中推理** (`gpt-5.5-medium`)
  - 标签: 多轮对话, 多模态, 长上下文, 联网搜索
  - 格式: `openai`
  - 端点: `/v1/chat/completions`
  - 简介: GPT-5.5-medium 是 OpenAI 推出的中等推理强度版本，在质量、速度、成本之间达到优雅平衡，适合日常推理任务。

### 图像模型 (image)

共 17 个模型

- **Nano Banana Pro** (`gemini-3-pro-image-preview`)
  - 标签: 文生图, 图生图, 4k, 高清, 香蕉
  - 简介: 谷歌2025年最新超高清图像模型，拥有目前最强的文字渲染能力，擅长生成8K分辨率的微距摄影、皮肤质感与复杂排版设计。

- **Nano Banana 2** (`gemini-3.1-flash-image-preview`)
  - 标签: 文生图, 图生图, 4k, 高清, 香蕉
  - 简介: 谷歌最新高效图像模型，Nano Banana Pro的高速版本，针对速度和高用量场景优化。支持联网搜索生图、Google图片搜索接地、512px快速预览，新增1:4/4:1/1:8/8:1超宽比例。

- **GPT Image 2** (`gpt-image-2`)
  - 标签: 文生图, 图生图
  - 简介: OpenAI 最新一代图像生成模型，语义理解与细节表现更强，支持文生图与图生图。

- **GPT Image 2 官转** (`gpt-image-2-guan`)
  - 标签: 文生图
  - 简介: OpenAI 官方直连通道，按 token 精准计费、多退少补；文生图与多图参考编辑兼备，画质与稳定性全面拉满。

- **VIDU Image 2** (`vidu-image-2`)
  - 标签: 文生图, 多图参考, 图片编辑, 1K/2K/3K, Vidu官方
  - 简介: Vidu 官方 reference2image：模型 viduimage-2，支持文生图、多图参考与图片编辑；最高 3K，质量与画幅可调。

- **即梦 5.0** (`doubao-seedream-5-0-260128`)
  - 标签: 文生图, 图生图, 多图融合, 联网搜索生图, 3k, 即梦
  - 简介: 字节跳动即梦AI图像生成模型，基于Seedream-5.0架构，支持多图融合、高清3K输出、联网搜索生图，文字和人脸细节表现出色

- **Midjourney** (`mj_imagine`)
  - 标签: 文生图, 图生图
  - 简介: Midjourney 是全球最火的 AI 图像生成模型，以极高的艺术性和美感著称。擅长生成电影级概念艺术、插画、海报、氛围图，支持通过垫图控制风格。

- **GPT Image 1.5** (`gpt-image-1.5-all`)
  - 标签: 文生图, 图生图
  - 简介: 集成在ChatGPT中的最新绘图引擎，拥有行业第一的语义理解力，无需复杂提示词即可听懂大白话逻辑，精准还原画面细节。

- **grok-4.2-image** (`grok-4.2-image`)
  - 标签: 文生图, 图生图
  - 简介: xAI旗下的新一代图像生成模型，基于Grok 4.2架构，相比4.1版本在画质细节和语义理解上进一步提升。支持文生图和图片编辑，每次固定返回2张图片，19种尺寸可选。

- **即梦 4.5** (`doubao-seedream-4-5-251128`)
  - 标签: 文生图, 图生图
  - 简介: 字节跳动即梦AI图像生成模型，基于Seedream-4.5架构，支持多图融合、高清4K输出，文字和人脸细节表现出色

- **万相 2.7 图像** (`wan2.7-image`)
  - 标签: 文生图, 图生图
  - 别名: 通义万相
  - 简介: 阿里云通义万相2.7图像模型，支持文生图和图像编辑，支持最高4K分辨率输出，拥有卓越的中文语义理解能力和思考增强模式，生成质量行业领先。

- **grok-4.1-image** (`grok-4.1-image`)
  - 标签: 文生图, 图生图
  - 简介: xAI旗下的图像生成模型，基于Grok 4.1架构，支持文生图和图片编辑，每次生成固定返回2张图片，提供19种尺寸选择，覆盖正方形、横版、竖版多种场景。

- **可灵-V3-Omni** (`kling-v3-omni`)
  - 标签: 文生图, 图生图, 多图融合, 4k, 高清
  - 简介: 快手可灵第三代 Omni 图像模型，支持文生图和多图融合，最多10张参考图，支持4K超高清输出，主体特征一致性强，适合多角色场景融合和风格迁移。

- **千问-image-max** (`qwen-image`)
  - 标签: 文生图, 图像编辑, 多图融合
  - 简介: 阿里云千问图像模型，支持文生图和图像编辑两种模式。文生图擅长复杂文本渲染、多行布局和图文混排；图像编辑支持最多3张参考图的多图融合，可精确修改图内文字、增删物体、迁移风格。

- **万相 2.6 图像** (`wan2.6-image`)
  - 标签: 文生图, 图生图
  - 别名: 通义万相
  - 简介: 阿里云通义万相2.6图像模型，支持图像编辑和多图参考生成，拥有业界领先的中文语义理解能力，可实现风格迁移、主体一致性生成等高级功能。

- **可灵 o1** (`kling-image-o1`)
  - 标签: 文生图, 图生图
  - 简介: 快手推出的旗舰级全能绘图模型，首创“生成”与“编辑”深度融合引擎。支持自然语言像素级改图，无需手动遮罩即可局部重构；拥有工业级的一致性控制，单次可融合10张参考图特征，是IP设计与商业海报创作的“所思即所得”神器。

- **可灵-V3** (`kling-v3`)
  - 标签: 文生图, 图生图, 高清
  - 简介: 快手可灵第三代图像生成模型，支持文生图和图生图两种模式，1K标清/2K高清可选，8种画面比例覆盖各类场景需求，光影质感和细节表现力全面升级。

### 视频模型 (video)

共 39 个模型

- **VIDU-音乐MV** (`vidu-mv`)
  - 标签: 音乐MV, 图生视频, 音频驱动, 对口型, 自动字幕, 540P/720P/1080P, 多画幅, 按秒计费
  - 简介: Vidu 音乐MV：上传 1～7 张人物/风格参考图 + 1 个音频（首尾合成1～180秒）即可生成一条有叙事、有画面的 MV，支持 540p/720p/1080p 多画幅 + 可选对口型 + 自动字幕，按实际成片秒数计费。

- **Sora-2 官转版** (`sora-2`)
  - 标签: 文生视频, 图生视频, 稳定
  - 简介: OpenAI Sora-2 稳定版，高质量视频生成，直接接入的官方接口，价格会比基础版稍贵，但是基本上保证100%成功率，且生成后的质量更高。

- **grok-video-3** (`grok-video-3`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 1080p, 高清
  - 简介: Grok 推出的首帧参考图视频模型，专注于高效的图生视频体验。支持生成 6 秒及 10 秒时长的 720P 分辨率视频，具备极快的响应速度。它能将静态图像瞬间转化为流畅的动态影像，是短视频创作者快速验证灵感与获取素材的便捷工具。

- **即梦 3.5 Pro** (`doubao-seedance-1-5-pro-251215`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 首尾帧, 有声视频, 1080p, 高清
  - 简介: 字节跳动即梦团队推出的高质量视频生成模型，支持音画同生，可生成带有环境音、动作音、背景音乐的有声视频，画质细腻流畅。

- **SD 2.0 参考生** (`kwvideo-v2-ref`)
  - 标签: 文生视频, 图生视频, 有声视频, 参考生视频, 即梦, 720p, 高清
  - 别名: Seedance, 即梦
  - 简介: 字节跳动即梦团队推出的旗舰级视频生成模型 Seedance 2.0，支持多图参考生视频，上传 1~9 张参考图，模型智能融合风格、元素和构图生成新视频。自动生成有声视频，4~15秒灵活时长，标准/快速双版本可选。按官方 Token 计费。

- **grok-video-3-plus** (`grok-video-3-plus`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 1080p, 高清
  - 简介: Grok 推出的 Plus 级视频生成模型，支持 10/15/20/25 秒多种时长，覆盖 16:9、9:16、3:2、2:3、1:1 全比例，适合社交媒体和创意短片场景。

- **可灵-Omni 参考生** (`kling-v3-omni-cankao`)
  - 标签: 文生视频, 图生视频, 有声视频, 参考生视频, 高清
  - 简介: 可灵 V3 Omni 参考生模式，支持纯文生视频或上传1-7张参考图片，AI参考图片风格/内容智能分镜生成有声视频，标准/高品质双模式，5-15秒。

- **快乐马-参考生** (`happyhorse-r2v`)
  - 标签: 参考生视频, 多主体融合, character指代, 角色一致, 多宽高比, 720P, 1080P, 阿里云百炼
  - 简介: 阿里百炼 HappyHorse 参考生（R2V）：可把多张参考图中的角色/物件融合进同一段短片。支持 720P/1080P、约 3–15 秒、多种画面比例（如 16:9、9:16）。

- **快乐马-视频编辑** (`happyhorse-video-edit`)
  - 标签: 视频编辑, 指令改片, 换装, 风格迁移, 可选参考图, 720P, 1080P, 阿里云百炼
  - 简介: 阿里百炼 HappyHorse 视频编辑：以一段已有视频为主线，可选用参考图锚定物体或人物外形，再配合自然语言完成风格迁移、换装、局部改写等；支持 720P/1080P，成片时长与原片相关。⚠️ 特殊计费：按「输入视频时长 + 输出视频时长」合计扣费，例如输入6秒+输出8秒=按14秒计，与常规仅按输出时长计费的视频模型不同，下单前请留意。

- **快乐马-文生视频** (`happyhorse-t2v`)
  - 标签: 文生视频, 物理真实, 运镜连贯, 720P, 1080P, 多比例, 阿里云百炼
  - 简介: 阿里百炼 HappyHorse（文生视频）：纯文本描述即可生成运动连贯、光影自然的短视频；支持 720P/1080P、3–15 秒多档时长，5 种宽高比自由选择。

- **veo3.1** (`veo3.1`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 首尾帧, 高清
  - 简介: 谷歌推出的高可控性视频模型，凭借独特的“首尾帧控制”技术（补全起始与结束画面）和精准运镜指令，能生成自带背景音乐的专业级视频。

- **SD 2.0 首尾帧** (`kwvideo-v2`)
  - 标签: 文生视频, 图生视频, 有声视频, 首尾帧, 即梦, 高清
  - 别名: Seedance, 即梦
  - 简介: 字节跳动即梦团队推出的旗舰级视频生成模型 Seedance 2.0，全球第一梯队超级多模态视频生成。支持文生视频、首帧图生视频、首尾帧三种模式，自动生成有声视频，4~15秒灵活时长，标准/快速双版本可选。按官方 Token 计费。

- **Vidu Q3 参考生** (`viduq3-cankaosheng`)
  - 标签: 参考生视频, 1080p, 多图参考, 有声视频, 高清
  - 简介: Vidu Q3 参考生视频模型，上传1-7张参考图片，AI以图中主体为参考生成主体一致的有声视频。当前版本为漫剧做了针对优化，支持音画同出，最高1080P分辨率，4-16秒时长可选。

- **快乐马-首帧** (`happyhorse-i2v`)
  - 标签: 图生视频, 首帧驱动, 物理真实, 运镜连贯, 720P, 1080P, 阿里云百炼
  - 简介: 阿里百炼 HappyHorse（图生视频）：以单张图片作首帧与视觉锚点，结合提示词生成运动连贯、光影自然的短视频；仅支持带图生成。支持 720P/1080P、3–15 秒多档时长，画幅随首帧自适应。

- **Pix V6 首尾帧** (`pixverse-v6-shouweizhen`)
  - 标签: 文生视频, 图生视频, 有声视频, 首尾帧, 高清
  - 简介: PixVerse V6 多模态视频生成，支持文生视频、图生视频、首尾帧三种模式。不传图=文生视频，1张图=首帧生视频，2张图=首尾帧，自动生成有声视频，支持360P-1080P分辨率，3-15秒。

- **可灵-动作控制 V3** (`kling-motion-control-v3`)
  - 标签: 动作控制, 视频生成, 高清
  - 简介: 可灵AI动作控制V3模型，基于V3引擎升级，通过上传参考图像和动作视频，让图片中的人物按照视频中的动作运动。支持std标准模式和pro高品质模式，可选择保留视频原声，人物朝向可与图片或视频一致。

- **veo3.1-4K高清** (`veo3.1-4k`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 首尾帧, 4K, 高清
  - 简介: Google Veo 3.1 4K 超清画质、逻辑级首尾帧控制、电影级运镜与原生音频生成于一体的“全能 AI 导演”，它让高难度的视频创意实现了从“不可控”到“精准定制”的跨越。

- **可灵-Omni 视频参考** (`kling-v3-omni-videoref`)
  - 标签: 视频参考, 视频编辑, 高清
  - 简介: 可灵 V3 Omni 视频参考模式，上传参考视频 + 可选0-4张参考图，支持视频参考（参考运镜/风格生新视频）和视频编辑（指令修改原视频）两种玩法，按秒计费。

- **Vidu Q3** (`viduq3`)
  - 标签: 文生视频, 图生视频, 首帧参考图, 首尾帧, 有声视频, 1080p, 高清
  - 简介: Vidu 推出的 Q3 系列视频生成模型，支持文生视频、首帧图生视频、首尾帧过渡视频三种模式。内置音视频同步直出能力，生成的视频自带台词和音效。提供快速(turbo)和高质量(pro)两种版本，最长支持16秒视频生成。

- **可灵-Omni 首尾帧** (`kling-v3-omni-shouweizhen`)
  - 标签: 图生视频, 有声视频, 首尾帧, 高清
  - 简介: 可灵 V3 Omni 首尾帧模式，上传1张图作为首帧，或2张图作为首帧+尾帧，AI智能分镜生成有声视频，支持标准/高品质双模式，5-15秒。

- **SD 2.0 全能参考** (`kwvideo-v2-quannengcankao`)
  - 标签: 多模态参考, 图参视参音参, 即梦, 1080p, 高清
  - 别名: Seedance, 即梦
  - 简介: 字节跳动即梦团队推出的旗舰级多模态参考视频生成模型 Seedance 2.0，支持文本+图片+视频+音频任意组合参考输入（至少需 1 图或 1 视），智能融合多种素材生成高质量有声视频。输出支持 480p / 720p / 1080p，按官方 Token 按输出分辨率与是否含参考视频分档计费。

- **可灵-V3-video** (`kling-v3-video`)
  - 标签: 文生视频, 图生视频, 有声视频, 首尾帧, 高清
  - 简介: 快手可灵第三代视频生成模型，支持文生视频和图生视频，支持首尾帧控制，标准/高品质双模式，5-15秒有声视频，画质和运动表现全面升级。

- **万相 2.6 参考生** (`wan2.6-cankaosheng`)
  - 标签: 参考生视频, 1080p, 高清
  - 简介: 万相2.6官方版参考生视频模型，支持上传参考图片提取角色形象，生成单角色或多角色互动视频。提供快速和高质量两档画质，支持720P/1080P分辨率，最长可生成10秒视频。

- **Pix C1 参考生** (`pixverse-c1-cankaosheng`)
  - 标签: 文生视频, 参考生, 有声视频, 高清, 动态场景
  - 简介: PixVerse C1 视频生成模型，专为打斗、法术特效及高速运动等动态场景优化。支持文生视频和参考生两种模式，不传图自动走文生，传图自动走参考生（最多7张），自动生成有声视频，支持360P-1080P分辨率，1-15秒。

- **veo3.1-lite** (`veo3.1-lite`)
  - 标签: 文生视频, 图生视频, 首尾帧, 标清/4K
  - 简介: Google最新的高级人工智能模型，veo3.1 lite 模式，支持视频自动配套音频生成，质量高价格很低，性价比最高的选择。

- **可灵-动作控制** (`kling-motion-control`)
  - 标签: 动作控制, 视频生成
  - 简介: 可灵AI动作控制模型，通过上传参考图像和动作视频，让图片中的人物按照视频中的动作运动。支持std标准模式和pro高品质模式，可选择保留视频原声，人物朝向可与图片或视频一致。

- **VIDU-解说漫** (`vidu-jieshuoman`)
  - 标签: 解说漫剧, 剧本驱动, 多资产参考, 角色独立音色, TTS+对口型, 中英双语, 720P/1080P, 多画幅, 按秒计费
  - 简介: Vidu 官方「解说漫」：以剧本驱动的解说向漫剧视频模型，支持 3–10 个角色/场景/道具参考图与独立角色音色，自带中英 TTS 与对口型。720P/1080P 多画幅可选，按实际成片秒数计费；上游推理时间较长（最长约 10 小时），请耐心等待。

- **万相 2.7 参考生** (`wan2.7-cankaosheng`)
  - 标签: 文生视频, 参考生视频, 1080p, 高清
  - 简介: 阿里云万相2.7旗舰参考生视频模型，支持文本生视频和参考生视频两种模式。纯文本自动走文生视频(t2v)，上传参考图片或视频后自动切换为参考生视频(r2v)。支持720P/1080P分辨率，最长15秒视频生成。

- **可灵-数字人** (`kling-avatar-image2video`)
  - 标签: 数字人, 视频生成
  - 简介: 可灵AI数字人模型，通过上传数字人参考图和音频文件，让图片中的人物开口说话。支持std标准模式和pro高品质模式，可定义数字人动作、情绪及运镜等。

- **万相-视频换人** (`wan2.2-animate-mix`)
  - 标签: 视频换人
  - 简介: 万相2.2视频换人模型，上传一张人物图片和一段参考视频，AI将视频中的人物替换为图片中的人物，支持标准模式和专业模式。无需输入提示词。

- **Pix C1 首尾帧** (`pixverse-c1-shouweizhen`)
  - 标签: 图生视频, 有声视频, 首尾帧, 高清, 动态场景
  - 简介: PixVerse C1 视频生成模型，专为打斗、法术特效及高速运动等动态场景优化。支持首帧生视频和首尾帧两种模式，自动生成有声视频，支持360P-1080P分辨率，1-15秒。

- **万相 2.7 首尾帧** (`wan2.7-shouweizhen`)
  - 标签: 图生视频, 首尾帧, 1080p, 高清
  - 简介: 阿里云万相2.7旗舰图生视频模型，支持首帧/首尾帧两种模式。上传1张图片自动走首帧生视频，上传2张图片自动走首尾帧生视频。支持720P/1080P分辨率，最长15秒视频生成。

- **Vidu Q2 参考生** (`viduq2-cankaosheng`)
  - 标签: 参考生视频, 1080p, 高清, 有声视频
  - 简介: Vidu Q2 参考生视频模型，上传1-7张参考图片，AI以图中主体为参考生成风格一致的高质量视频。提供标准版（细节丰富）和高质量（支持视频参考、视频编辑）两种模式，最高1080P分辨率，5-10秒时长可选。

- **Pix V5.6 参考生** (`pixverse-v5.6-r2v`)
  - 标签: 参考生视频, 多图参考, 有声视频, 图生视频, 高清
  - 简介: PixVerse V5.6 参考生视频模型，支持上传1-7张参考图片，AI参考图片中的角色、风格、场景融合生成有声视频。支持360P至1080P分辨率，5-10秒时长可选。

- **Pix V5.6 首尾帧** (`pixverse-v5.6-shouweizhen`)
  - 标签: 文生视频, 图生视频, 有声视频, 首尾帧, 高清
  - 简介: PixVerse V5.6 多模态视频生成，支持文生视频、图生视频、首尾帧三种模式。不传图=文生视频，1张图=首帧生视频，2张图=首尾帧，自动生成有声视频，支持360P-1080P分辨率，5-10秒。

- **万相 2.6 首帧** (`wan2.6-shouzheng`)
  - 标签: 图生视频, 文生视频, 有声视频, 1080p, 首帧参考图, 高清
  - 简介: 万相2.6官方版图生视频模型，支持首帧图片驱动和纯文本生成两种模式，自动生成有声视频。提供快速和高质量两档画质，支持720P/1080P分辨率，最长可生成15秒电影级视频。

- **可灵 2.6 Pro** (`kling-v2-6`)
  - 标签: 文生视频, 图生视频, 有声视频, 1080p, 高清
  - 简介: 快手推出的“物理世界模拟器”视频旗舰可灵 2.6，支持文生视频与单图生视频。凭借卓越的 Transformer 架构，仅需一张参考图即可生成符合真实物理规律的 1080p 电影级视频。它在光影一致性与大幅度运动表现上实现了质的飞跃，是能将静态灵感瞬间转化为动态大片的“光影造梦引擎”。

- **万相 2.7 视频续写** (`wan2.7-xuxie`)
  - 标签: 视频续写, 视频转视频, 1080p, 高清
  - 简介: 阿里云万相2.7视频续写模型，支持首段视频续写和首段视频+尾帧续写两种模式。上传视频片段自动续写，可选添加尾帧图片引导视频方向。支持720P/1080P分辨率，最长15秒视频生成。

- **海螺 2.3** (`hailuo-2.3`)
  - 标签: 文生视频, 图生视频, 1080p, 高清
  - 简介: 海螺AI是MiniMax推出的视频生成模型，2.3版本在动作自然度、物理真实感和指令遵循能力上实现重大突破。提供标准版和极速版两种选择，极速版价格更优惠，适合批量创作。

### 音频/TTS/音乐模型 (audio)

共 6 个模型

- **海螺 语音克隆 2.8** (`speech-2.8`)
  - 标签: 语音克隆, 文字转语音, 音色复刻, 多语言
  - 简介: MiniMax 海螺语音克隆模型，支持上传音频复刻你的专属音色，首次激活后永久有效。提供 HD 高清和 Turbo 极速两种合成质量，支持语速、语调、情绪、音效等精细调节，适用于有声读物、配音、播客等场景。

- **海螺 音乐生成 2.5+** (`music-2.5+`)
  - 标签: 音乐生成, 歌词生成, AI作曲, 纯音乐
  - 简介: MiniMax 海螺音乐生成增强模型，支持歌曲生成和纯音乐生成两种模式。歌曲模式下输入歌词和风格描述生成完整歌曲；纯音乐模式下仅需风格描述即可生成无人声的纯器乐音乐。支持 AI 歌词创作、14+ 曲式结构标签精准控制、多风格混合，带来专业级编曲混音。

- **豆包 语音合成 2.0** (`doubao-tts-2.0`)
  - 标签: 文字转语音, 多音色, 多情感, 多语言
  - 简介: 火山引擎豆包语音合成2.0，支持上百种精品音色，支持多情感、多语种、长文本合成（最大10万字符），音质自然流畅。

- **Gemini-3.1-TTS** (`gemini-3.1-flash-tts-preview`)
  - 标签: 文字转语音, 多音色, 多语言
  - 简介: Google Gemini 3.1 Flash 原生文字转语音模型，支持30种预置音色和24种语言，支持双人对话，支持自然语言描述语气，速度更快、价格更优。

支持各种长文本朗读，有声小说，多人对话，爽得飞起。

- **海螺 音乐生成 2.5** (`music-2.5`)
  - 标签: 音乐生成, 歌词生成, AI作曲
  - 简介: MiniMax 海螺音乐生成模型，输入歌词和风格描述即可生成完整歌曲。支持 AI 歌词创作、14+ 曲式结构标签精准控制、多风格混合（流行/摇滚/电子/古典/爵士等），带来高保真人声演唱和专业级编曲混音。

- **Gemini-2.5-TTS** (`gemini-2.5-pro-preview-tts`)
  - 标签: 文字转语音, 多音色, 多语言
  - 简介: Google Gemini 原生文字转语音模型，支持30种预置音色和24种语言，支持双人对话，支持自然语言描述语气，跟剪映的AI配音不是一个级别的东西。

支持各种长文本朗读，有声小说，多人对话，爽得飞起。

## 调用指南

### 语言模型调用

万象Ai 支持三种格式的语言模型调用：

- **OpenAI 格式**: `POST /v1/chat/completions`
- **Anthropic 格式**: `POST /v1/messages`
- **Gemini 格式**: `POST /v1beta/models/{model}:{action}`

调用前先用 `GET /v1/skills/models?type=chat` 查询模型，根据返回的 `api_format` 和 `api_endpoint` 选择对应格式。

### 媒体生成调用

- 先 `GET /v1/media/models` 查询可用媒体模型
- 再 `POST /v1/media/generate` 提交生成任务
- 用 `GET /v1/skills/task-status` 轮询任务状态
