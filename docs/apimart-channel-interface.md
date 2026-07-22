# APIMart 渠道接口清单

> 来源：`relay/channel/task/apimart/adaptor.go`、`relay/image_handler.go`、`relay/relay_adaptor.go`、`constant/channel.go`  
> 渠道类型：`ChannelTypeAPIMart = 61`  
> 默认 Base URL：`https://api.apimart.ai`  
> 适配器：`relay/channel/task/apimart/adaptor.go`  
> 任务模式：异步任务（图像/视频均走 task 异步流程）

---

## 1. 下游应用端接口

下游应用只需要调用 new-api 提供的 OpenAI 兼容接口，new-api 内部会根据 `model` 路由到 APIMart。

### 1.1 图片生成

```http
POST /v1/images/generations
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

```json
{
  "model": "gpt-image-2",
  "prompt": "a patriotic merchandise ad, baseball cap",
  "size": "1024x1024",
  "aspect_ratio": "1:1",
  "n": 1,
  "image_urls": ["https://your-cdn.com/ref.jpg"]
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 目前 APIMart 对应 `gpt-image-2` |
| `prompt` | string | ✅ | 图片描述 |
| `size` | string | ❌ | 像素尺寸，如 `1024x1024`，直接透传给 APIMart |
| `aspect_ratio` | string | ❌ | 比例，如 `1:1`、`16:9`、`9:16`，直接透传 |
| `n` | int | ❌ | 生成数量，默认 1 |
| `image_urls` | array | ❌ | 参考图 URL 数组（http/https/asset://） |
| `image` | string / array | ❌ | 参考图（OpenAI 兼容字段） |
| `reference_images` | array | ❌ | 参考图（multipart/URL） |

**同步响应示例：**

```json
{
  "created": 1760759307,
  "data": [
    {
      "url": "https://heharse.cloud/api/image-proxy/xxx.png",
      "revised_prompt": "..."
    }
  ]
}
```

**异步响应示例（部分场景会包装成任务）：**

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "image.generation",
  "model": "gpt-image-2",
  "status": "queued",
  "progress": "0%",
  "created_at": 1760759307
}
```

**异步轮询：**

```http
GET /v1/images/tasks/{task_id}
Authorization: Bearer <token>
```

---

### 1.2 视频生成

```http
POST /v1/videos/generations
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

```json
{
  "model": "veo3.1-fast",
  "prompt": "a cat running on grass",
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "duration": 6,
  "image_urls": ["https://your-cdn.com/ref.jpg"]
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | `veo3.1-fast`、`veo3.1-quality`、`veo3.1-lite`、`Omni-Flash-Ext`、`grok-imagine-1.5-video-apimart`（兼容 `grok-imagine-1.5-video-ext`） |
| `prompt` | string | ✅ | 视频描述 |
| `aspect_ratio` | string | ❌ | 比例，默认 `16:9` |
| `resolution` | string | ❌ | `480p`/`720p`/`1080p`/`4k`，默认 `720p` |
| `duration` | int | ❌ | 时长（秒），默认 6 |
| `image_urls` | array | ❌ | 参考图 URL 数组 |
| `video_urls` | array | ❌ | 视频参考 URL 数组 |
| `generation_type` | string | ❌ | 如 `reference`（多图融合） |
| `enable_gif` | bool | ❌ | 是否生成 GIF |
| `official_fallback` | bool | ❌ | 是否官方回退 |

**提交响应示例：**

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video.generation",
  "model": "veo3.1-fast",
  "status": "queued",
  "progress": "0%",
  "created_at": 1760759307
}
```

**异步轮询：**

```http
GET /v1/videos/{task_id}
Authorization: Bearer <token>
```

---

### 1.3 multipart/form-data 图片上传（图生图）

APIMart 支持直接用 multipart 上传参考图：

```http
POST /v1/images/generations
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**表单字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `prompt` | string | 图片描述 |
| `model` | string | 模型名 |
| `size` | string | 尺寸 |
| `aspect_ratio` | string | 比例 |
| `image` | file | 参考图文件（优先） |
| `ref_images` | file[] | 多参考图文件 |
| `images` | file[] | 兼容字段 |
| `files` | file[] | 兼容字段 |

new-api 会：
1. 把二进制文件保存到 `uploads/` 目录
2. 生成公网 URL（基于 `UPLOADS_PUBLIC_URL` 环境变量）
3. 把 URL 放到 `image_urls` 中传给 APIMart

**环境变量：**

```bash
UPLOADS_PUBLIC_URL=https://your-domain.com/uploads/
```

---

## 2. new-api 内部调用 APIMart 的上游接口

| 功能 | 上游 APIMart 接口 | 方法 |
|------|------------------|------|
| 图片生成 | `https://api.apimart.ai/v1/images/generations` | POST |
| 视频生成 | `https://api.apimart.ai/v1/videos/generations` | POST |
| 任务查询 | `https://api.apimart.ai/v1/tasks/{task_id}` | GET |

---

## 3. APIMart 上游请求体格式

### 3.1 图片生成请求

```json
{
  "model": "gpt-image-2",
  "prompt": "...",
  "n": 1,
  "size": "1024x1024",
  "aspect_ratio": "1:1",
  "resolution": "1k",
  "actual_image_count": 1,
  "effective_resolution": "1K",
  "image_urls": ["https://your-domain.com/uploads/ref_xxx.jpg"]
}
```

**字段转换规则：**

- `size`：直接透传 `req.Size`
- `aspect_ratio`：直接透传 `req.AspectRatio` 或 `metadata.aspect_ratio`
- `resolution`：从 `metadata.resolution` 取，默认 `1k`
- `n`：从 `metadata.n` 取，默认 1
- `actual_image_count`：等于 `n`
- `effective_resolution`：`1k` → `1K`，`2k` → `2K`，`4k` → `4K`
- `image_urls`：从 `reference_images` → `images` → `image_urls` → `image` 依次取

### 3.2 视频生成请求

```json
{
  "model": "veo3.1-fast",
  "prompt": "...",
  "duration": 6,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "image_urls": ["..."],
  "generation_type": "reference"
}
```

**字段转换规则：**

- `duration`：默认 6，若传 `video_urls` 则忽略 duration
- `aspect_ratio`：默认 `16:9`
- `resolution`：默认 `720p`，支持 `480p`/`720p`/`1080p`/`4k`
- `image_urls`：参考图 URL
- `video_urls`：视频参考 URL（用于视频编辑）
- `generation_type`：从 `metadata.generation_type` 取；若多图则自动设为 `reference`
- `enable_gif`：从 `metadata.enable_gif` 取
- `official_fallback`：从 `metadata.official_fallback` 取

---

## 4. APIMart 上游响应格式

### 4.1 提交响应

```json
{
  "code": 200,
  "data": [
    {
      "status": "submitted",
      "task_id": "upstream_task_xxx"
    }
  ]
}
```

### 4.2 查询响应

```json
{
  "code": 200,
  "data": {
    "id": "upstream_task_xxx",
    "status": "completed",
    "progress": 100,
    "result": {
      "images": [
        {
          "url": ["https://generated-url/xxx.png"],
          "expires_at": 1783402110
        }
      ],
      "videos": [
        {
          "url": ["https://generated-url/xxx.mp4"],
          "expires_at": 1783402110
        }
      ]
    }
  }
}
```

**状态映射：**

| APIMart 状态 | new-api 状态 | Progress |
|-------------|-------------|----------|
| `pending` / `processing` | `in_progress` | 按上游 progress |
| `completed` | `success` | `100%` |
| `failed` | `failure` | `100%` |
| `cancelled` | `failure` | `100%` |

---

## 5. 支持的模型列表

```go
[]string{
    // Image
    "gpt-image-2",
    // Video
    "veo3.1-fast",
    "veo3.1-quality",
    "veo3.1-lite",
    "Omni-Flash-Ext",
}
```

---

## 6. 关键代码路径

| 文件 | 职责 |
|------|------|
| `relay/channel/task/apimart/adaptor.go` | APIMart 适配器：构造请求、解析响应、任务轮询 |
| `relay/relay_adaptor.go` | `GetTaskAdaptor` 中按 `ChannelTypeAPIMart` 返回 `apimart.TaskAdaptor` |
| `relay/image_handler.go` | `ImageHelper` 中识别 task image channel，走 `RelayTaskSubmit` |
| `controller/relay.go` | 视频生成入口 `RelayVideo` / 任务提交入口 `RelayTask` |
| `relay/relay_task.go` | 异步任务提交与轮询框架 |
| `service/async_image.go` | 异步图像任务注册与结果缓存 |
| `model/task.go` | 任务模型与数据库操作 |

---

## 7. 注意事项

1. **参考图必须是 URL**：APIMart 只接受 `http://` / `https://` / `asset://` URL。new-api 会自动把 multipart 上传的文件或 `data:image/...base64` 转成本地 URL。
2. **图片生成也是异步**：APIMart 的图片生成通过 task 机制提交，返回 `task_id` 给下游。
3. **Base URL 可配置**：在渠道配置里填写 APIMart 的 API 地址，默认是 `https://api.apimart.ai`。
4. **密钥**：在 new-api 后台「渠道」里填 APIMart 的 API Key。
5. **debug 日志**：APIMart 适配器会写 `/tmp/apimart_debug.log`（Release 模式下调试用）。

---

## 8. 下游 curl 示例

### 图片生成

```bash
curl -X POST "https://heharse.cloud/api/v1/images/generations" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "a premium patriotic merchandise ad, baseball cap on oak pedestal",
    "size": "1024x1024",
    "image_urls": ["https://your-cdn.com/ref.jpg"]
  }'
```

### 视频生成

```bash
curl -X POST "https://heharse.cloud/api/v1/videos/generations" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "veo3.1-fast",
    "prompt": "a cat running on grass",
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "duration": 6
  }'
```

### 任务轮询

```bash
curl "https://heharse.cloud/api/v1/videos/task_xxx" \
  -H "Authorization: Bearer sk-xxx"
```
