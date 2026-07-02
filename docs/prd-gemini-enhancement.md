# Gemini 渠道增强 PRD

## 1. 项目信息

- **Language**: 中文
- **Programming Language**: Go（new-api 后端）
- **Project Name**: `gemini_enhancement`
- **原始需求**: 在 new-api 中全面升级 Google Gemini 渠道能力，覆盖图片生成、Veo 视频生成、Omni Flash 视频生成、文件上传、多模态 Chat 五项能力，并以 OpenAI 兼容接口对外暴露。

## 2. 产品目标

让 new-api 的 Gemini 渠道在 OpenAI 兼容接口下完整支持图片、视频、音频、文档等多模态输入与生成能力，并复用现有异步任务与文件上传架构，降低后续维护成本。

## 3. 用户故事

- As a 开发者，我希望能通过 `/v1/images/generations` 调用 Gemini 图片生成模型，以便在现有 OpenAI 兼容客户端中直接生成图片。
- As a 视频创作者，我希望能通过 `/v1/videos/generations` 调用 Gemini Veo 3.1 异步生成视频，并通过任务轮询获取结果，以便将视频生成集成到工作流中。
- As a 多模态应用开发者，我希望能通过 `/v1/chat/completions` 发送图片、视频、音频、PDF 文件进行对话，以便构建富媒体 AI 应用。
- As a 平台管理员，我希望能通过 `/v1/files` 上传本地文件到 Google Files API 并拿到 `file_uri`，以便在多模态 Chat 和媒体生成中复用。
- As a 渠道运营者，我希望 Gemini 图片/视频/聊天的模型、计费、配额管理能与现有 new-api 渠道体系保持一致，以便统一控制成本。

## 4. 需求池

### 4.1 P0（必须完成）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P0-1 | 图片生成接口对齐 | `/v1/images/generations` → Gemini `generateContent` 图片生成 | 支持 `gemini-3.1-flash-image`、`gemini-3-pro-image` 等模型；支持 `responseModalities: [TEXT, IMAGE]`；支持 `responseFormat.image`（`aspectRatio`、`imageSize`）；参考图最多 14 张；支持 `grounding`/`dynamic_retrieval` 等高级参数；返回 OpenAI 兼容 `b64_json` 或 `url` |
| P0-2 | Veo 视频生成 | `/v1/videos/generations` → Veo 3.1 `predictLongRunning` | 支持 `veo-3.1-generate-preview`；复用 `relay/channel/task` 异步任务+轮询；支持文生视频、图生视频、参考图、首帧/尾帧；支持 `aspectRatio`、`resolution`、`durationSeconds`；返回 OpenAI 兼容任务 id/视频 URL |
| P0-3 | 文件上传 OpenAI 兼容 | `POST /v1/files` → Google Files API | 支持本地文件上传和 resumable upload；返回 OpenAI 兼容 file id 及 Google `file_uri`；单文件最大 2 GB、项目总 20 GB、48 小时有效期；复用 `dto/file.go`、`relay/file_handler.go` 架构 |
| P0-4 | 多模态 Chat 增强 | `/v1/chat/completions` → Gemini `generateContent` | 支持 `image_url`（url/base64/file_uri）、`video_url`（url/file_uri）、`input_audio`（url/base64/file_uri）、`file`（PDF，url/file_uri）；支持 `thinkingConfig`、`system_instruction`；复用现有 `relay-gemini.go` 适配器 |
| P0-5 | 渠道模型与模型映射配置 | 渠道类型 `ChannelTypeGemini` 与 `ChannelTypeVeo` | 模型列表支持新模型；图片/视频生成模型通过 `model` 参数识别；新增模型映射与模型部署关系在后台可配置 |

### 4.2 P1（应该完成）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P1-1 | Omni Flash 视频生成 | `/v1/videos/generations` → Gemini Omni Flash Interactions API | 支持 `gemini-omni-flash-preview`；支持文生视频、图生视频、reference 视频；支持 stateful editing（session 维持）；返回 OpenAI 兼容任务/视频结果 |
| P1-2 | 异步任务下载托管 | 媒体生成结果视频/图片下载 | 是否将 Google 返回的临时媒体 URL 转存为 new-api 内部可访问 URL，需与现有 task 架构一致 |
| P1-3 | 错误码与重试策略 | 图片/视频生成与文件上传 | 明确 Google 配额、内容政策、文件大小超限等错误码，并返回 OpenAI 兼容 `error` 结构；支持可配置重试 |

### 4.3 P2（可选）

| 编号 | 需求 | 范围 | 验收标准 |
|------|------|------|---------|
| P2-1 | 文件列表与删除 | `GET /v1/files`、`DELETE /v1/files/{file_id}` | 对接 Google Files API 的 list/get/delete，便于用户管理已上传文件 |
| P2-2 | 多模态消息流式输出 | `/v1/chat/completions` stream | 对 Gemini 生成的多模态内容（图片+文本）支持流式返回，按 OpenAI 流式格式组织 |
| P2-3 | 图片生成批量请求 | `/v1/images/generations` | 支持一次请求生成多张图片（n > 1），并在 Gemini 限制范围内做拆分或报错 |

## 5. 接口映射

### 5.1 图片生成：`/v1/images/generations`

| OpenAI 字段 | Gemini 字段 | 说明 |
|------------|------------|------|
| `model` | `model` | 例如 `gemini-3.1-flash-image` |
| `prompt` | `contents[0].parts[0].text` | 提示文本 |
| `response_format` / `size` | `generationConfig.responseModalities` + `responseFormat.image` | 映射到 `aspectRatio`（如 `1:1`, `16:9`, `3:4`, `4:3`）和 `imageSize`（如 `1024x1024`） |
| `n` | 单次生成张数 | Gemini 原生可能一次返回一张或多张，需按模型能力限制处理 |
| 参考图（OpenAI 自定义字段） | `contents[0].parts[].inlineData` / `fileData` | 支持 url、base64、file_uri；最多 14 张 |
| `quality` / `style` | `generationConfig` 扩展字段 | 如 `grounding`、`dynamic_retrieval` 透传 |

返回结构：OpenAI `ImagesResponse`（`data[].b64_json` / `url`）。

### 5.2 视频生成：`/v1/videos/generations`

| OpenAI 字段 | Gemini Veo 字段 | Gemini Omni Flash 字段 | 说明 |
|------------|------------------|----------------------|------|
| `model` | `model` = `veo-3.1-generate-preview` | `model` = `gemini-omni-flash-preview` | 根据模型名分发到 Veo 或 Omni Flash 适配器 |
| `prompt` | `instances[0].prompt` | `input[].text` | 文本提示 |
| 参考图/首帧/尾帧 | `instances[0].image` / `instances[0].referenceImages` | `input[].image` / `reference` | 支持 url/base64/file_uri |
| `size` / `aspect_ratio` | `parameters.aspectRatio` | `config.aspectRatio` | 如 `16:9`、`9:16` |
| `quality` / `resolution` | `parameters.resolution` | `config.resolution` | 如 `720p`、`1080p`、`4k` |
| `duration` | `parameters.durationSeconds` | `config.durationSeconds` | 如 `4`、`6`、`8` |
| 扩展视频 | `parameters` 扩展 | `input.reference` | 支持输入已有视频/前次结果作为 extension |

异步流程：

1. 接收 `POST /v1/videos/generations`。
2. 创建 new-api 内部 task（复用 `relay/channel/task`）。
3. 调用 Google 对应 endpoint：`predictLongRunning`（Veo）或 `interactions.create`（Omni Flash）。
4. 轮询 `GET /operations/{name}` 或 `GET /interactions/{id}`，直到完成。
5. 下载/转存媒体文件，返回 OpenAI 兼容视频 URL 或 `b64_json`。

返回结构：OpenAI 兼容视频生成响应（含 `task_id`、最终 `video_url`）。

### 5.3 文件上传：`POST /v1/files`

| OpenAI 字段 | Google Files API 字段 | 说明 |
|------------|----------------------|------|
| `file`（multipart） | 文件二进制 | 支持本地文件上传 |
| `purpose` | `display_name` / 透传 | 保留 OpenAI 字段，用于文件用途标记 |
| `filename` | `file.display_name` | 展示名称 |
| 返回 `id` | `file.name` | 内部 file id，如 `files-xxx` |
| 返回 `file_uri` | `file.uri` | 供 Chat/图片/视频生成使用，48 小时有效 |
| 返回 `bytes` / `created_at` | `file.size_bytes` / `file.create_time` | 元信息透传 |

上传流程：

1. 接收 `POST /v1/files` multipart。
2. 判断文件大小：<100 MB 可走简单上传；≥100 MB 或 PDF>50 MB 走 resumable upload（`X-Goog-Upload-Protocol: resumable`）。
3. 调用 `POST /upload/v1beta/files`。
4. 返回 OpenAI 兼容文件对象，并在 `file_uri` 字段中暴露 Google 文件 URI。

### 5.4 多模态 Chat：`/v1/chat/completions`

| OpenAI content 类型 | Gemini `contents` 类型 | 说明 |
|---------------------|----------------------|------|
| `text` | `text` part | 普通文本消息 |
| `image_url`（url / base64） | `inlineData` / `fileData` | 支持 url、base64、file_uri |
| `video_url`（url / file_uri） | `fileData` | 视频使用 Google file_uri 或先上传 |
| `input_audio`（url / base64 / file_uri） | `fileData` + mimeType | 音频输入 |
| `file`（PDF，url / file_uri） | `fileData` | 文档输入 |
| `system` message | `system_instruction` | 提取 system role 并映射 |
| `thinkingConfig`（自定义字段） | `generationConfig.thinkingConfig` | 透传 thinking budget 等参数 |
| 流式 | `streamGenerateContent` | 按需支持 |

返回结构：OpenAI `ChatCompletion` 或 `ChatCompletionChunk`（stream）。

## 6. UI 设计草稿（管理后台）

- **渠道配置页**：在 Gemini 渠道表单中新增模型选择（图片/视频/聊天模型），支持 `model` 参数映射。
- **模型管理页**：新增 `gemini-3.1-flash-image`、`gemini-3-pro-image`、`veo-3.1-generate-preview`、`gemini-omni-flash-preview` 等模型元数据，配置倍率、Quota、是否启用。
- **文件管理页（可选）**：展示已上传文件列表、file_uri、过期时间，支持删除。
- **任务管理页**：媒体生成任务（图片/视频）展示任务状态、轮询进度、结果 URL。

## 7. 待确认问题

1. **模型列表与部署名**：最终确认图片/视频/多模态 Chat 需要支持的具体模型部署名，例如 `gemini-3.1-flash-image` 在 Google 侧是否即为 `models/gemini-3.1-flash-image`？
2. **计费方式**：图片/视频生成按张、按秒、按 token 还是按请求计费？Veo 不同分辨率（720p/1080p/4k）是否差异化计费？Omni Flash 是否按视频输出时长计费？
3. **Base64 处理策略**：图片/音频/视频 base64 输入的大小限制是多少？是否超过阈值后自动转 Google Files API 上传？base64 输出是否压缩/转码？
4. **Omni Flash stateful session**：`gemini-omni-flash-preview` 的 stateful editing 是否需要 new-api 维护 session id？若需要，session 生命周期、并发数、过期时间如何定义？
5. **Files API 的 file_uri 有效期**：Google 文件 48 小时自动删除，new-api 是否需要在数据库中记录文件元数据并提示过期？是否需要提供下载/转存机制？
6. **图片生成响应格式**：当 `responseModalities` 同时包含 TEXT 和 IMAGE 时，OpenAI 的 `/v1/images/generations` 是否只取 IMAGE 部分？TEXT 部分如何透传或丢弃？
7. **异步任务结果存储**：Veo/Omni Flash 生成的视频结果 URL 是否需要在 new-api 本地转存？如果转存，存储路径、CDN、过期策略如何设计？
8. **grounding / dynamic_retrieval 透传**：这些高级参数是否直接透传给 Gemini，还是需要在 new-api 层做校验或默认值？
9. **视频生成参考图数量**：Veo 3.1 图片输入最多 3 张参考图 + 首帧/尾帧，new-api 是否在前端做数量校验？
10. **错误码映射**：Google 侧的内容安全拦截、配额耗尽、文件格式不支持等错误，如何映射到 OpenAI 兼容错误码？

---

*文档版本：v1.0*
*作者：产品经理（software-product-manager）*
*日期：2026-07-23*
