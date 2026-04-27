# New-API 渠道接口文档

## 更新日志
| 日期 | 变更内容 |
|------|---------|
| 2026-04-27 | 任务日志延迟修复：提交任务后立即在后台可见（预插入占位机制，状态流转：`queued` → `in_progress` → `success`/`failure`） |
| 2026-04-25 | 渠道测试修复：`/uapi/` 路径支持 RelayModeVideoSubmit，GeminiGen 渠道可在后台直接测试；渠道名称统一为 GeminiGen；新增 `/uapi/v1/upload_images` 图片上传接口；`file_urls` 文本字段转发修复 |
| 2026-04-23 | `ref_images` 支持三种格式（multipart 文件/base64 data URL/HTTP URL）；`nano-banana-2` 图生图验证通过 |
| 2026-04-22 | `/uapi/` 通道修复完成，视频和图片接口全部验证通过；新增 seedance-2-remix/omni 视频模型 |
| 2026-04-21 | 初始文档 |

## 目录
1. [OpenAI 兼容接口](#1-openai-兼容接口)
2. [GeminiGen 渠道](#2-geminigen-渠道)
3. [分镜助手接入方式](#3-分镜助手接入方式)
4. [通用参数说明](#4-通用参数说明)

---

## 1. OpenAI 兼容接口

### 1.1 基础信息
- **Base URL**: `https://heharse.cloud`
- **认证方式**: `Authorization: Bearer {API_KEY}`

### 1.2 聊天补全 `/v1/chat/completions`
```bash
curl -X POST https://heharse.cloud/v1/chat/completions \
  -H "Authorization: Bearer {API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "你好"}],
    "max_tokens": 1000
  }'
```

**支持的模型**: `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `claude-3-5-sonnet`, `claude-3-haiku`, `gemini-2.0-flash`, `gemini-1.5-pro` 等

### 1.3 图片生成 `/v1/images/generations`
```bash
curl -X POST https://heharse.cloud/v1/images/generations \
  -H "Authorization: Bearer {API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "dall-e-3",
    "prompt": "一个可爱的机器人",
    "size": "1024x1024",
    "n": 1
  }'
```

### 1.4 文本补全 `/v1/completions`
```bash
curl -X POST https://heharse.cloud/v1/completions \
  -H "Authorization: Bearer {API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo-instruct",
    "prompt": "The first president of the United States was"
  }'
```

### 1.5 Embeddings `/v1/embeddings`
```bash
curl -X POST https://heharse.cloud/v1/embeddings \
  -H "Authorization: Bearer {API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-3-small",
    "input": "The food was delicious"
  }'
```

### 1.6 语音转文字 `/v1/audio/transcriptions`
```bash
curl -X POST https://heharse.cloud/v1/audio/transcriptions \
  -H "Authorization: Bearer {API_KEY}" \
  -F "file=@audio.mp3" \
  -F "model=whisper-1"
```

---

## 2. GeminiGen 渠道

### 2.1 基础信息
- **Base URL**: `https://heharse.cloud`
- **认证方式**: `Authorization: Bearer {API_KEY}`（`sk-` 前缀可选，自动处理）
- **渠道测试**: GeminiGen 渠道支持在后台管理界面直接点击"测试"按钮进行验证（测试请求走 `/uapi/` 端点）

### 2.2 视频生成

#### 提交任务: `/uapi/v1/video-gen/{model}`

```bash
# Veo 模型
curl -X POST https://heharse.cloud/uapi/v1/video-gen/veo \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A serene sunset over mountains with clouds" \
  -F "model=veo-3.1" \
  -F "resolution=720p"

# Grok 模型
curl -X POST https://heharse.cloud/uapi/v1/video-gen/grok \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A cat playing piano" \
  -F "model=grok-3" \
  -F "resolution=1080p"

# Seedance 模型
curl -X POST https://heharse.cloud/uapi/v1/video-gen/seedance \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=Dramatic ocean waves crashing on rocks" \
  -F "model=seedance-2" \
  -F "resolution=720p"

# Kling 模型
curl -X POST https://heharse.cloud/uapi/v1/video-gen/kling \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=Time-lapse of a flower blooming" \
  -F "model=kling" \
  -F "resolution=1080p"
```

**支持的视频模型**: `veo-3.1`, `veo-3.1-fast`, `veo-2`, `veo-3.1-lite`, `grok-3`, `grok-video`, `seedance-2`, `seedance-2-remix`, `seedance-2-omni`, `kling`

**默认时长与分辨率**:
| 模型 | 默认时长 | 支持分辨率 |
|------|---------|-----------|
| Veo 系列 | 8 秒 | 480p, 720p, 1080p |
| Grok 视频 | 6/10/15 秒 | 480p, 720p, 1080p |
| Seedance | 5 秒 | 480p, 720p, 1080p |
| Kling | 5/10/15/30 秒 | 480p, 720p, 1080p |

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | 是 | 视频描述文本 |
| model | string | 是 | 模型名称 |
| resolution | string | 否 | 分辨率: `480p`, `720p`, `1080p` |
| aspect_ratio | string | 否 | 宽高比: `16:9`(默认), `9:16`, `1:1` |
| ref_images | file | 否 | 参考图片（支持 1-3 张，frame 模式最多 2 张） |
| mode_image | string | 否 | 图片参考模式: `frame`(默认), `ingredient` |

**提交响应**:
```json
{
  "id": "task_abc123",
  "task_id": "task_abc123",
  "uuid": "上游任务的 UUID（用于轮询）",
  "object": "video",
  "model": "veo-3.1",
  "status": "queued",
  "progress": 0,
  "created_at": 1713000000
}
```

> **重要**：`task_id` 是 New-API 的任务 ID（用于接口调用），`uuid` 是上游任务的唯一标识（用于轮询）。旧版任务可能没有 `uuid`，New-API 会自动从提交响应中解析。

#### 查询任务: `/uapi/v1/video-gen/veo?task_id={task_id}`

```bash
curl "https://heharse.cloud/uapi/v1/video-gen/veo?task_id={task_id}" \
  -H "Authorization: Bearer {API_KEY}"
```

---

### 2.3 图片生成

#### 提交任务: `/uapi/v1/generate_image`

```bash
curl -X POST https://heharse.cloud/uapi/v1/generate_image \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A beautiful landscape with mountains and a lake" \
  -F "model=nano-banana-2" \
  -F "resolution=1K"
```

**支持的图片模型**: `nano-banana-pro`, `nano-banana-2`, `imagen-4`, `grok-image`, `meta-ai-image`

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | 是 | 图片描述文本 |
| model | string | 是 | 模型名称 |
| resolution | string | 否 | 分辨率: `1K`(默认), `2K`, `4K` |
| aspect_ratio | string | 否 | 宽高比: `1:1`(默认), `16:9`, `9:16`, `4:3`, `3:4` |
| style | string | 否 | 艺术风格: `Photorealistic`, `Anime General`, `3D Render`, `Illustration` 等 |
| output_format | string | 否 | 输出格式: `jpeg`(默认), `png` |
| ref_images | file | 否 | 参考图片（relay 转为上游 `files` 字段，支持 multipart file） |
| files | file | 否 | 参考图片（直接发给上游，支持 multipart file） |
| file_urls | string | 否 | 参考图片 URL（数组，上游自己下载） |

**提交响应**:
```json
{
  "id": "task_abc123",
  "task_id": "task_abc123",
  "uuid": "上游任务的 UUID（用于轮询）",
  "object": "video",
  "model": "nano-banana-2",
  "status": "queued",
  "progress": 0,
  "created_at": 1713000000
}
```

> **注意**：参考图可用 `files`（multipart 文件）或 `file_urls`（URL 文本）传给上游，两者可同时使用。`ref_images` 是客户端友好的别名，relay 会自动转为 `files` 字段。

#### 查询任务: `/uapi/v1/generate_image?task_id={task_id}`

```bash
curl "https://heharse.cloud/uapi/v1/generate_image?task_id={task_id}" \
  -H "Authorization: Bearer {API_KEY}"
```

#### 其他图片路径

```bash
# Grok 图片
curl -X POST https://heharse.cloud/uapi/v1/imagen/grok \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=An astronaut riding a horse" \
  -F "model=grok-image"

# Meta AI 图片
curl -X POST https://heharse.cloud/uapi/v1/meta_ai/generate \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A futuristic city at night" \
  -F "model=meta-ai-image"
```

---

### 2.4 图片上传: `/uapi/v1/upload_images`

将本地图片上传到服务器，返回公开访问的 CDN URL 列表，供后续 `file_urls` 字段使用。

```bash
curl -X POST https://heharse.cloud/uapi/v1/upload_images \
  -H "Authorization: Bearer {API_KEY}" \
  -F "images=@/path/to/image1.jpg" \
  -F "images=@/path/to/image2.jpg" \
  -F "images=@/path/to/image3.png"
```

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| images | file | 是 | 图片文件，支持多张（多个 `images` 字段） |

**响应**:
```json
{
  "urls": [
    "https://heharse.cloud/uploads/uuid1.jpg",
    "https://heharse.cloud/uploads/uuid2.png"
  ]
}
```

> **使用流程**：本地图片 → `POST /uapi/v1/upload_images` → 获得 URL 列表 → `POST /uapi/v1/generate_image` 的 `file_urls` 字段使用

### 2.5 JSON 图片上传: `/uapi/v1/upload_images/json`

不走 multipart，完全通过 JSON body 提交 base64 图片数据，避免部分 HTTP 客户端 multipart 兼容性问题。

```bash
curl -X POST https://heharse.cloud/uapi/v1/upload_images/json \
  -H "Authorization: Bearer {API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "images": [
      "data:image/png;base64,iVBORw0KGgo..."
    ]
  }'
```

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| images | array | 是 | 图片数据 URL 字符串，支持 `data:image/png;base64,...` 格式 |

**响应**:
```json
{
  "urls": [
    "https://heharse.cloud/uploads/uuid.png"
  ]
}
```

> 与 2.4 的 multipart 接口功能相同，推荐 Rust/Python 等需要绕开 multipart 的客户端使用。

---

### 3.1 推荐接入方式

#### 方式一: OpenAI 兼容接口（推荐）

```javascript
const config = {
  baseURL: 'https://heharse.cloud',
  apiKey: 'YOUR_API_KEY',
  model: 'gpt-4o'
}

async function generateStoryboard(prompt) {
  const response = await fetch(`${config.baseURL}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${config.apiKey}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model: config.model,
      messages: [
        { role: 'system', content: '你是一个专业的分镜头脚本助手。' },
        { role: 'user', content: prompt }
      ],
      max_tokens: 2000,
      temperature: 0.7
    })
  })
  return response.json()
}
```

#### 方式二: GeminiGen 图片生成（带参考图）

```javascript
async function generateImage(prompt, refImageFile, model = 'nano-banana-2') {
  const formData = new FormData()
  formData.append('prompt', prompt)
  formData.append('model', model)
  formData.append('resolution', '1K')
  if (refImageFile) {
    formData.append('ref_images', refImageFile)  // 支持多张
  }

  const response = await fetch(`${config.baseURL}/uapi/v1/generate_image`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${config.apiKey}` },
    body: formData
  })
  return response.json()  // 返回 { task_id: "task_xxx", status: "queued", ... }
}
```

#### 方式三: 任务轮询

```javascript
async function pollTask(taskId) {
  const resp = await fetch(`${config.baseURL}/uapi/v1/generate_image?task_id=${taskId}`, {
    headers: { 'Authorization': `Bearer ${config.apiKey}` }
  })
  return resp.json()  // { code: "success", data: { status, progress, result_url, data: {...} } }
}

async function waitForCompletion(taskId, interval = 5000) {
  while (true) {
    const result = await pollTask(taskId)
    const { status, progress, fail_reason, data } = result.data
    // 状态流转：queued（已提交）→ in_progress（处理中）→ SUCCESS / FAILURE
    if (status === 'SUCCESS') {
      return data?.generated_image?.[0]?.image_url || result.data.result_url
    }
    if (status === 'FAILURE') {
      throw new Error(fail_reason || '任务失败')
    }
    console.log(`状态: ${status}, 进度: ${progress}`)
    await new Promise(r => setTimeout(r, interval))
  }
}
```

### 3.2 Python SDK 示例

```python
import requests

class NewAPI:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url
        self.api_key = api_key
        self.headers = {'Authorization': f'Bearer {api_key}'}

    def chat(self, model: str, messages: list, **kwargs):
        response = requests.post(
            f'{self.base_url}/v1/chat/completions',
            headers=self.headers,
            json={'model': model, 'messages': messages, **kwargs}
        )
        return response.json()

    def generate_image(self, model: str, prompt: str,
                     resolution='1K', ref_images=None, **kwargs):
        data = {'model': model, 'prompt': prompt, 'resolution': resolution, **kwargs}
        files = {}
        if ref_images:
            # ref_images 支持三种格式：
            # 1. 本地文件路径：'ref.jpg' 或 ['ref.jpg'] -> 转为 files 字段
            # 2. base64 data URL：'data:image/png;base64,...'
            # 3. HTTP 图片 URL：'https://example.com/img.png' -> 转为 file_urls 字段
            if isinstance(ref_images, str):
                ref_images = [ref_images]
            for i, path in enumerate(ref_images):
                files[f'ref_images'] = open(path, 'rb')
        resp = requests.post(
            f'{self.base_url}/uapi/v1/generate_image',
            headers=self.headers,
            data=data,
            files=files or None
        )
        for f in files.values():
            f.close()
        return resp.json()

    def poll_task(self, task_id: str, interval=5, timeout=300):
        import time
        start = time.time()
        while time.time() - start < timeout:
            resp = requests.get(
                f'{self.base_url}/uapi/v1/generate_image?task_id={task_id}',
                headers=self.headers
            )
            data = resp.json().get('data', {})
            # 状态流转：queued -> in_progress -> SUCCESS / FAILURE
            if data['status'] == 'SUCCESS':
                return data['data']['generated_image'][0]['image_url']
            if data['status'] == 'FAILURE':
                raise Exception(data['fail_reason'])
            print(f"状态: {data['status']}, 进度: {data['progress']}")
            time.sleep(interval)
        raise TimeoutError('轮询超时')

# 使用示例
api = NewAPI('https://heharse.cloud', 'YOUR_API_KEY')
result = api.generate_image('nano-banana-2', '衣服变蓝', ref_images='ref.jpg')
url = api.poll_task(result['task_id'])
print(f"图片URL: {url}")

# 带参考图的完整流程（推荐）：
# 1. 上传本地图片获取 URL
upload_resp = requests.post(
    f'{self.base_url}/uapi/v1/upload_images',
    headers=self.headers,
    files=[('images', open('ref.jpg', 'rb')), ('images', open('ref2.jpg', 'rb'))]
)
urls = upload_resp.json()['urls']
# 2. 用 URL 调用图生图
for url in urls:
    result = api.generate_image('nano-banana-2', '衣服变蓝', file_urls=url)
```

---

## 4. 通用参数说明

### 4.1 认证

| 接口类型 | Header | 示例 |
|---------|--------|------|
| 所有接口 | `Authorization: Bearer {key}` | `Bearer sk-xxx` 或 `Bearer xxx` 均可 |

> relay 中间件自动处理 `sk-` 前缀。

### 4.2 错误码

| 错误码 | 说明 |
|--------|------|
| `401` | 认证失败，请检查 API Key |
| `429` | 请求过于频繁，请稍后重试 |
| `500` | 服务器内部错误 |
| `insufficient_user_quota` | 余额不足 |
| `model_not_found` | 模型未配置渠道 |
| `task_not_exist` | 任务不存在 |
| `fail_to_fetch_task` | 上游请求失败（如参考图文件无效） |
| `INVALID_FILE_CONTENT` | 参考图文件格式不正确（非图片） |
| `FILE_DOWNLOAD_FAILED` | `file_urls` 中的图片 URL 无法下载 |
| `INVALID_FILE_CONTENT` | 参考图文件格式不正确（非图片） |

### 4.3 任务状态流转

提交任务后，New-API 会**立即写入占位记录**，任务日志立即可见，随后由轮询器同步更新上游状态：

| 状态 | 说明 | 出现时机 |
|------|------|---------|
| `queued` | 已提交，等待上游处理 | 提交后立即出现 |
| `in_progress` | 上游正在处理中 | 上游开始生成后 |
| `success` | 生成完成 | 上游返回结果 |
| `failure` | 生成失败 | 上游报错或超时 |

> 带参考图（`ref_images`/`file_urls`）的上游 submit 调用本身耗时较长（5-15秒），但占位机制保证任务日志在**提交瞬间**即可见，不必等到上游响应。

### 4.4 通用响应格式

**提交成功**:
```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "nano-banana-2",
  "status": "queued",
  "progress": 0,
  "created_at": 1713000000
}
```

**轮询成功（图片）**:
```json
{
  "code": "success",
  "data": {
    "task_id": "task_xxx",
    "uuid": "上游 UUID",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://cdn.example.com/image.jpg",
    "data": {
      "type": "image",
      "reference_item": [{"media_type": "image", "thumbnail_url": "..."}],
      "generated_image": [{"image_url": "https://...", "status": 2}]
    }
  }
}
```

**轮询成功（视频）**:
```json
{
  "code": "success",
  "data": {
    "task_id": "task_xxx",
    "uuid": "上游 UUID",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://cdn.example.com/video.mp4",
    "data": {
      "type": "video",
      "generated_video": [{"video_url": "https://...", "duration": 8.0, "status": 2}]
    }
  }
}
```

> 视频 URL 由 Cloudflare CDN 直接提供，非视频代理地址。

**错误响应**:
```json
{
  "error": {
    "message": "错误信息",
    "type": "invalid_request_error",
    "code": "invalid_api_key"
  }
}
```

---

## 5. 模型列表

### 5.1 文本模型

| 模型名 | 渠道 | 说明 |
|--------|------|------|
| `gpt-4o` | OpenAI | 最新 GPT-4 模型 |
| `gpt-4o-mini` | OpenAI | 高性价比 GPT-4 |
| `claude-3-5-sonnet` | Anthropic | Claude 3.5 Sonnet |
| `claude-3-haiku` | Anthropic | 快速 Claude 3 |
| `gemini-2.0-flash` | Gemini | Google Gemini 2.0 |
| `gemini-1.5-pro` | Gemini | Google Gemini 1.5 Pro |
| `deepseek-chat` | DeepSeek | DeepSeek 聊天模型 |

### 5.2 图片模型

| 模型名 | 渠道 | 说明 |
|--------|------|------|
| `nano-banana-pro` | GeminiGen | Gemini 3 Pro 高质量图片 |
| `nano-banana-2` | GeminiGen | Gemini 3 Flash 快速图片 |
| `imagen-4` | GeminiGen | Google Imagen 4 |
| `grok-image` | GeminiGen | Grok 图片生成 |
| `meta-ai-image` | GeminiGen | Meta AI 图片 |
| `dall-e-3` | OpenAI | OpenAI 图片生成 |

### 5.3 视频模型

| 模型名 | 渠道 | 说明 |
|--------|------|------|
| `veo-3.1` | GeminiGen | Google Veo 3.1 高质量 |
| `veo-3.1-fast` | GeminiGen | Veo 3.1 快速版 |
| `veo-2` | GeminiGen | Google Veo 2 |
| `veo-3.1-lite` | GeminiGen | Veo 3.1 Lite（带音频） |
| `grok-3` | GeminiGen | Grok 视频生成 |
| `grok-video` | GeminiGen | Grok 视频 |
| `seedance-2` | GeminiGen | 即梦视频基础版 |
| `seedance-2-remix` | GeminiGen | 即梦视频 remix 版 |
| `seedance-2-omni` | GeminiGen | 即梦视频 omni 版 |
| `kling` | GeminiGen | 快手可灵 |

---

## 6. Windows 客户端注意事项

使用 Rust/Python 等语言在 Windows 上调用 New-API 时，如遇到 `fail_to_fetch_task` 或连接问题，请确保 **禁用系统代理**：

```rust
// Rust (reqwest)
Client::builder()
    .no_proxy()
    .build()
```

```python
# Python (requests)
session = requests.Session()
session.trust_env = False  # 忽略系统代理
```

---


