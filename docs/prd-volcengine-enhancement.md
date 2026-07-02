# VolcEngine 渠道增强 PRD

> **项目**：new-api 火山方舟（VolcEngine / Ark）渠道能力增强  
> **PRD 版本**：v1.0  
> **编写日期**：2026-07-02  
> **建议落盘路径**：`F:\new api\docs\prd-volcengine-enhancement.md`  
> **负责人**：产品经理 · 许清楚（software-product-manager）

---

## 1. 项目背景与目标

new-api 已支持火山方舟（VolcEngine）基础文本对话、TTS、Embedding、Rerank 与图片生成（Seedream）的透传能力。随着豆包 Seedream 5.0/4.5/4.0 以及多模态对话模型的能力升级，下游用户希望 new-api 能够完整对齐火山方舟 `/api/v3` 接口的最新能力，重点补齐以下三项：

1. **图片生成增强**：支持 `2K/3K/4K` 分辨率、多图参考（image array）、`sequential_image_generation` 组图生成等 Seedream 原生参数。
2. **文件上传**：接入火山方舟 `/api/v3/files`，支持二进制文件与 URL 两种方式上传，返回 `file_id` 与可访问 URL。
3. **多模态 Chat**：增强 VolcEngine 聊天适配器，让 `image_url`（url/file_id/base64）、`video_url`（url/file_id）、`input_audio`（url/file_id/base64）、`file`（PDF，url/file_id）等多模态消息格式能够正确透传并响应。

**核心目标**：在不破坏现有 OpenAI 兼容接口的前提下，让 new-api 的 VolcEngine 渠道具备与火山方舟官方 API 对齐的“图片生成 + 文件上传 + 多模态对话”能力，使下游客户端可以沿用现有 `/v1/images/generations`、`/v1/chat/completions` 与新增 `/v1/files` 调用形态即可使用火山方舟完整能力。

---

## 2. 协作边界

- **项目范围**：仅修改 `relay/channel/volcengine` 包及其相关 DTO/常量，必要时新增通用工具函数，**不引入新的渠道类型**。
- **受影响模块**：
  - `relay/channel/volcengine/*`
  - `dto/openai_request.go`（如需扩展通用消息类型）
  - `dto/openai_image.go`（如需扩展图片请求字段）
  - 路由层可能涉及 `/v1/files` 的文件上传转发逻辑
- **不在本 PRD 范围内**：
  - 前端 UI 改造（如模型选择、参数表单）
  - 计费/价格表配置
  - 新渠道接入（如豆包视频生成 task 通道已单独存在，不在本需求内）
  - 火山方舟 Bot / Claude / Gemini 兼容路径的改动（保持原有逻辑）
- **目标仓库**：`F:\new api\`
- **下游技术约束**：Go 1.22+（以 `go.mod` 为准），使用现有 `common.Marshal` / `common.Unmarshal` 工具，遵循现有 `channel.Adaptor` 接口。

---

## 3. 术语与引用

| 术语 | 说明 |
|------|------|
| Seedream | 火山方舟图片生成模型系列，如 `doubao-seedream-5-0-260128`、`doubao-seedream-4-5-251128`、`doubao-seedream-4-0-250828` |
| 2K/3K/4K | Seedream 支持的分辨率标识，分别对应约 2048×、3072×、4096× 级别的短边尺寸 |
| image array | 请求体中 `image` 字段可为字符串（单图）或字符串数组（多图参考） |
| `sequential_image_generation` | Seedream 组图生成开关，取值为 `disabled` / `auto` |
| `file_id` | 火山方舟 Files API 返回的文件 ID，如 `file-20251018114827-6zgrb` |
| TOS URI | 火山对象存储资源标识，格式 `tos://<bucket>/<prefix>/<file_name>` |

**参考文档**：
- 火山方舟 Seedream 图片生成教程：https://www.volcengine.com/docs/82379/1824121
- 火山方舟 Files API 上传文件：https://www.volcengine.com/docs/82379/1870405
- 火山方舟 Chat Completions API：https://www.volcengine.com/docs/82379/1494384

---

## 4. 用户故事

### 4.1 图片生成增强

- 作为开发者，我希望在调用 `/v1/images/generations` 时传入 `size: "2K"`、`image: ["url1", "url2"]`、`sequential_image_generation: "auto"`，从而使用 Seedream 5.0 的多图参考和组图生成能力。
- 作为开发者，我希望 `response_format: "b64_json"` 时，new-api 能把火山方舟返回的图片 URL 自动下载并转为 base64 返回，保持与 OpenAI 接口一致。

### 4.2 文件上传

- 作为开发者，我希望通过 `/v1/files` 上传本地二进制文件或提供公网 URL，获得 `file_id` 与可访问 URL，用于后续多模态对话或图片生成参考。
- 作为开发者，我希望上传视频文件时能透传 `preprocess_configs`（如视频 fps），保持与火山方舟官方能力一致。

### 4.3 多模态 Chat

- 作为开发者，我希望在 `/v1/chat/completions` 中发送 `image_url`（支持 url / base64 / file_id）、`video_url`（url / file_id）、`input_audio`（url / base64 / file_id）、`file`（PDF url / file_id）等消息类型，让火山方舟多模态模型理解并回复。
- 作为开发者，我希望这些多模态消息能被正确映射到火山方舟 `/api/v3/chat/completions` 的 content parts 格式，不丢失字段。

---

## 5. 需求池

### 5.1 P0 — 必须完成

| 编号 | 需求 | 归属模块 | 验收标准 |
|------|------|----------|----------|
| P0-1 | 图片生成请求透传 `size` 为 `2K/3K/4K` 标识 | 图片生成 | Seedream 请求体正确携带 `size` 字段，响应正常返回图片 URL |
| P0-2 | 图片生成支持单图/多图参考 `image` 字符串数组 | 图片生成 | 单图保持字符串，多图转为数组透传，不丢失顺序 |
| P0-3 | 图片生成支持 `sequential_image_generation` 与 `sequential_image_generation_options` | 图片生成 | 字段原样透传，组图生成返回多张图片 |
| P0-4 | 图片生成支持 `response_format: "b64_json"` 时下载转 base64 | 图片生成 | 返回 OpenAI 标准 `b64_json` 字段 |
| P0-5 | 接入 `/api/v3/files` 二进制文件上传 | 文件上传 | 支持 `multipart/form-data` 上传，返回 `file_id` 与 URL |
| P0-6 | 接入 `/api/v3/files` URL 上传 | 文件上传 | 支持 `url` 参数，返回 `file_id` 与 URL |
| P0-7 | 多模态 Chat 支持 `image_url`（url / base64 / file_id） | 聊天适配 | 能被火山方舟模型正确识别并回复 |
| P0-8 | 多模态 Chat 支持 `video_url`（url / file_id） | 聊天适配 | 能被火山方舟视频理解模型正确识别 |
| P0-9 | 多模态 Chat 支持 `input_audio`（url / base64 / file_id） | 聊天适配 | 能被火山方舟音频理解模型正确识别 |
| P0-10 | 多模态 Chat 支持 `file`（PDF，url / file_id） | 聊天适配 | 能被火山方舟文档理解模型正确识别 |

### 5.2 P1 — 强烈建议完成

| 编号 | 需求 | 归属模块 | 验收标准 |
|------|------|----------|----------|
| P1-1 | 图片生成请求体透传其他 Seedream 扩展字段（如 `output_format`、`watermark`、`prompt_optimizer` 等） | 图片生成 | 未在 OpenAI 标准字段中定义但用户传入的字段能原样透传 |
| P1-2 | 文件上传支持 `purpose`、`preprocess_configs`、`tos` 等可选参数 | 文件上传 | 参数正确映射到火山方舟请求 |
| P1-3 | 多模态消息中的 base64 数据能自动检测 MIME 类型并补齐 data URL | 聊天适配 | 火山方舟收到 `data:image/png;base64,...` 或类似格式 |
| P1-4 | 为新增能力补充单元测试，覆盖请求转换与响应转换 | 测试 | 核心分支有测试覆盖 |

### 5.3 P2 — 可选增强

| 编号 | 需求 | 归属模块 | 验收标准 |
|------|------|----------|----------|
| P2-1 | 图片生成失败时返回更清晰的错误信息（解析火山方舟错误体） | 图片生成 | 错误码与 message 正确透传 |
| P2-2 | 文件上传后返回 OpenAI 标准 `FileObject` 格式（包含 `object`、`created_at`、`status` 等） | 文件上传 | 与 OpenAI `/v1/files` 响应格式一致 |

---

## 6. 详细需求说明

### 6.1 图片生成增强

#### 6.1.1 请求字段映射

new-api 接收 OpenAI 标准 `/v1/images/generations` 请求（`dto.ImageRequest`），在 VolcEngine adapter 内转换为火山方舟 `/api/v3/images/generations` 请求。

| OpenAI 字段 | VolcEngine 字段 | 说明 |
|------------|-----------------|------|
| `model` | `model` | 需先经过 `MapSeedreamImageModel` 映射为真实 Model ID |
| `prompt` | `prompt` | 直接透传 |
| `size` | `size` | 支持 `2K`、`3K`、`4K` 以及传统像素尺寸 |
| `n` | `n` | 生成数量 |
| `response_format` | `response_format` | `url` / `b64_json` |
| `quality` | `quality` | 直接透传 |
| `style` | `style` | 直接透传 |
| `image` | `image` | 新增：单图为字符串，多图为字符串数组 |
| — | `sequential_image_generation` | 新增：取值为 `disabled` / `auto` |
| — | `sequential_image_generation_options` | 新增：对象，如 `{"max_images": 4}` |
| — | `output_format` | 新增：Seedream 输出格式，如 `png`、`jpeg` |
| — | `watermark` | 新增：布尔 |
| 其他 `Extra` 字段 | 原样透传 | 保证未来字段无需改代码即可兼容 |

> 说明：VolcEngine 图片生成的 `image` 字段在官方文档中既支持单图字符串，也支持多图字符串数组。new-api 当前仅做模型映射后直接透传，需要显式支持 `image` 数组及 `sequential_image_generation` 参数。

#### 6.1.2 响应处理

- 当 `response_format` 为 `url`（默认）时，直接把火山方舟返回的 URL 列表包装为 OpenAI `ImageResponse`。
- 当 `response_format` 为 `b64_json` 时，需要把 URL 下载后转为 base64 字符串写入 `b64_json` 字段。
- 响应结构示例：

```json
{
  "created": 1760759307,
  "data": [
    {"url": "https://..."},
    {"b64_json": "..."}
  ]
}
```

#### 6.1.3 模型别名

保持现有 `SeedreamImageModelAliases` 映射不变，但需确认新增 Seedream 模型版本时能够平滑扩展。本需求不新增模型别名，沿用现有映射。

---

### 6.2 文件上传

#### 6.2.1 接口形态

new-api 对外暴露 `/v1/files`（与 OpenAI 兼容），转发到火山方舟 `/api/v3/files`。

#### 6.2.2 请求方式

支持两种上传方式：

1. **二进制文件上传**：`multipart/form-data`，字段名 `file`，可选 `purpose`、`preprocess_configs` 等。
2. **URL 上传**：`multipart/form-data` 或 `application/json`（以官方文档为准，优先使用 multipart），字段名 `url`，可选 `purpose`、`preprocess_configs` 等。

#### 6.2.3 请求字段映射

| 客户端字段 | VolcEngine 字段 | 必填 | 说明 |
|-----------|-----------------|------|------|
| `file` | `file` | 与 `url` 二选一 | 二进制文件 |
| `url` | `url` | 与 `file` 二选一 | HTTP/HTTPS URL 或 TOS URI |
| `purpose` | `purpose` | 否，默认 `user_data` | 文件用途 |
| `preprocess_configs` | `preprocess_configs` | 否 | 预处理配置，如视频 fps |
| `tos` | `tos` | 否 | 对象存储目标 |
| `expire_at` | `expire_at` | 否 | 过期时间（UTC 时间戳） |

#### 6.2.4 响应字段

火山方舟返回标准的 `file` 对象，new-api 应原样透传或转换为 OpenAI 标准格式：

```json
{
  "object": "file",
  "id": "file-20251018114827-6zgrb",
  "purpose": "user_data",
  "filename": "demo.mp4",
  "bytes": 695110,
  "mime_type": "video/mp4",
  "created_at": 1760759307,
  "expire_at": 1761364107,
  "status": "processing",
  "preprocess_configs": { "video": { "fps": 0.3 } }
}
```

#### 6.2.5 路由与鉴权

- 使用 VolcEngine 渠道的 `Authorization: Bearer {api_key}` 鉴权。
- 请求 URL 基于 `info.ChannelBaseUrl` + `/api/v3/files`。
- 二进制上传时保持 `Content-Type: multipart/form-data`，需正确处理文件头与边界。

---

### 6.3 多模态 Chat

#### 6.3.1 接口形态

new-api 接收标准 `/v1/chat/completions` 请求，其中 `messages` 的 `content` 为数组，包含多种 content part。VolcEngine adapter 在 `ConvertOpenAIRequest` 中把消息转换为火山方舟 `/api/v3/chat/completions` 支持的格式。

#### 6.3.2 支持的消息类型与映射

| OpenAI content part 类型 | 支持来源 | VolcEngine 格式 | 说明 |
|-------------------------|---------|----------------|------|
| `text` | `text` 字段 | 直接透传 | 文本消息 |
| `image_url` | `url` 字符串或 `{"url": "..."}` 对象 | 图片 URL / base64 / file_id | 支持 `http(s)`、`data:image/*;base64`、火山 `file_id` |
| `video_url` | `video_url` 字符串或 `{"url": "..."}` 对象 | 视频 URL / file_id | 支持 `http(s)`、火山 `file_id` |
| `input_audio` | `{"data": "base64", "format": "mp3"}` 或 `{"url": "..."}` / `{"file_id": "..."}` | 音频数据 | 支持 base64、URL、file_id |
| `file` | `{"file_id": "..."}` 或 `{"filename": "...", "file_data": "base64"}` 或 `{"url": "..."}` | PDF 文件 | 支持 URL、file_id、base64（PDF） |

#### 6.3.3 数据源转换规则

- **URL**：以 `http://` 或 `https://` 开头，直接透传。
- **base64 图片**：格式为 `data:image/{png|jpeg|webp};base64,{...}`，直接透传。
- **base64 音频**：需要补齐 `data:audio/{format};base64,{...}` 或按火山方舟要求格式透传。
- **file_id**：以 `file-` 开头，直接透传。
- **TOS URI**：格式 `tos://...`，直接透传。

#### 6.3.4 字段转换示例

OpenAI 请求：

```json
{
  "model": "doubao-seed-1-6-thinking-250715",
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "描述这张图片" },
        { "type": "image_url", "image_url": { "url": "https://example.com/a.png" } },
        { "type": "input_audio", "input_audio": { "data": "...", "format": "mp3" } }
      ]
    }
  ]
}
```

转换后火山方舟请求：

```json
{
  "model": "doubao-seed-1-6-thinking-250715",
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "描述这张图片" },
        { "type": "image_url", "image_url": { "url": "https://example.com/a.png" } },
        { "type": "input_audio", "input_audio": { "data": "...", "format": "mp3" } }
      ]
    }
  ]
}
```

> 火山方舟 Chat API 对 OpenAI 多模态 content part 格式高度兼容，核心工作为：
> 1. 识别 `file_id` 并透传；
> 2. 识别 base64 并确保前缀正确；
> 3. 对不支持的字段给出明确错误。

---

## 7. 非功能需求

- **兼容性**：所有改动不能破坏现有 VolcEngine TTS、Embedding、Rerank、文本对话功能。
- **错误处理**：当火山方舟返回非 2xx 响应时，应解析其错误体并返回 OpenAI 标准错误格式。
- **日志**：关键转换节点（图片请求转换、文件上传、多模态消息转换）应记录 `LogDebug` 日志，便于排查。
- **性能**：图片 URL 转 base64 时使用连接池与超时，避免长时间阻塞。
- **安全性**：文件上传大小限制由 new-api 全局配置控制，不单独放宽。
- **代码风格**：遵循现有 Go 代码风格，使用 `common.Marshal` / `common.Unmarshal`，避免引入新的第三方依赖。

---

## 8. 验收标准

### 8.1 图片生成

1. 请求 `size: "4K"`、`image: ["url1", "url2"]`、`sequential_image_generation: "auto"` 时，火山方舟能正常返回多张图片。
2. `response_format: "b64_json"` 时，返回的 `data` 数组中每个元素包含 `b64_json` 字段，且内容可解码为图片。
3. 单图参考（`image` 为字符串）与多图参考（`image` 为数组）均能正常工作。
4. 新增字段（如 `output_format`、`watermark`）能原样透传。

### 8.2 文件上传

1. 通过 `/v1/files` 上传本地图片/视频文件，返回 `file_id` 与 `status`。
2. 通过 `/v1/files` 上传公网 URL，返回 `file_id` 与 `status`。
3. 上传视频时透传 `preprocess_configs`，返回体中包含对应字段。
4. 错误场景（如文件过大、URL 不可访问）返回清晰错误信息。

### 8.3 多模态 Chat

1. 包含 `image_url`（url / base64 / file_id）的聊天请求能被火山方舟多模态模型正常响应。
2. 包含 `video_url`（url / file_id）的聊天请求能被火山方舟视频理解模型正常响应。
3. 包含 `input_audio`（url / base64 / file_id）的聊天请求能被火山方舟音频理解模型正常响应。
4. 包含 `file`（PDF url / file_id）的聊天请求能被火山方舟文档理解模型正常响应。
5. 混合多种模态的单条消息能正确转换并透传。

---

## 9. 待确认问题

1. 火山方舟 Chat API 的 `file` 多模态部分是否要求 `file_id` 必须来自 `/api/v3/files` 上传后的文件，还是也支持直接 URL？（建议实现时两者都支持）
2. 文件上传接口在 new-api 路由层是否已有 `/v1/files` 占位（目前为 `RelayNotImplemented`），是否由本需求直接实现该路由转发？
3. `sequential_image_generation_options` 是否需要严格校验 `max_images` 范围？
4. 多模态消息中的 base64 是否需要限制最大尺寸（如 10MB）？

---

## 10. 风险与依赖

- **风险 1**：火山方舟 API 文档中多模态 content part 的精确字段未完全公开，实现时需要参考官方返回或 SDK 示例进行对齐。
- **风险 2**：`/v1/files` 路由目前为 `RelayNotImplemented`，需要确认路由层如何区分渠道类型并调用对应 adapter 的文件上传能力。
- **风险 3**：图片 URL 转 base64 涉及网络下载，可能因上游 URL 失效或超时导致失败，需要完善错误处理与重试策略。
- **依赖**：本需求依赖于 `relay/channel/volcengine` 现有结构，需先由架构师完成设计与任务分解。

---

## 11. 附录：参考示例

### 11.1 Seedream 多图参考请求

```bash
curl https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "doubao-seedream-5-0-260128",
    "prompt": "将图1的服装换为图2的服装",
    "image": [
      "https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_imagesToimage_1.png",
      "https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_5_imagesToimage_2.png"
    ],
    "sequential_image_generation": "disabled",
    "size": "2K",
    "output_format": "png",
    "watermark": false
  }'
```

### 11.2 Files API 上传示例

```bash
curl https://ark.cn-beijing.volces.com/api/v3/files \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -F 'purpose=user_data' \
  -F 'file=@/root/projects/ark/ark_video.mp4' \
  -F 'preprocess_configs[video][fps]=0.3'
```

### 11.3 多模态 Chat 请求示例

```json
{
  "model": "doubao-seed-1-6-thinking-250715",
  "messages": [
    {
      "role": "user",
      "content": [
        { "type": "text", "text": "总结这份 PDF 的内容" },
        { "type": "file", "file": { "file_id": "file-20251018114827-6zgrb" } }
      ]
    }
  ]
}
```
