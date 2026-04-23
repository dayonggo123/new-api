# GeminiGen 渠道 API 接口文档

## 0. 重要更新（2026-04-23）

`ref_images` 参考图现已支持三种格式：multipart 文件、base64 data URL、HTTP 图片 URL。图片图生图功能（nano-banana-2 等）验证通过，上游成功接收参考图并返回 `reference_item`。

### 快速测试

```bash
# 视频生成
curl -X POST https://heharse.cloud/uapi/v1/video-gen/veo \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A cat playing piano" \
  -F "model=veo-3.1"

# 图片生成（带 HTTP URL 参考图）
curl -X POST https://heharse.cloud/uapi/v1/generate_image \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=衣服变红" \
  -F "model=nano-banana-2" \
  -F "ref_images=https://example.com/ref.jpg"

# 轮询（用 task_id）
curl "https://heharse.cloud/uapi/v1/video-gen/veo?task_id={task_id}" \
  -H "Authorization: Bearer {API_KEY}"
```

---

## 1. 概述

本文档描述 GeminiGen 渠道（ChannelType=58）的完整 API 接口，供下游应用对接使用。

### 1.1 基础信息

| 项目 | 值 |
|------|-----|
| 渠道类型 | `ChannelTypeVeo = 58` |
| Base URL | `https://heharse.cloud` |
| 认证方式 | `Authorization: Bearer {API_KEY}` |
| 内容类型 | `multipart/form-data`（推荐）或 `application/json` |

### 1.2 可用接口

| 功能 | 提交接口 | 轮询接口 |
|------|---------|---------|
| 视频生成 | `POST /uapi/v1/video-gen/{model}` | `GET /uapi/v1/video-gen/{model}?task_id={task_id}` |
| 图片生成 | `POST /uapi/v1/generate_image` | `GET /uapi/v1/generate_image?task_id={task_id}` |

> **注意**：轮询时使用提交返回的 `task_id`（不要用 `uuid`）。

### 1.3 支持模型

**视频模型：**

| 模型名 | 上游路径 | 默认时长 | 分辨率 |
|--------|---------|---------|--------|
| `veo-3.1` | `/uapi/v1/video-gen/veo` | 8s | 480p / 720p / 1080p |
| `veo-3.1-fast` | `/uapi/v1/video-gen/veo` | 8s | 480p / 720p / 1080p |
| `veo-2` | `/uapi/v1/video-gen/veo` | 8s | 480p / 720p / 1080p |
| `veo-3.1-lite` | `/uapi/v1/video-gen/veo` | 8s（带音频） | 480p / 720p / 1080p |
| `grok-3` | `/uapi/v1/video-gen/grok` | 6/10/15s | 480p / 720p / 1080p |
| `grok-video` | `/uapi/v1/video-gen/grok` | 6/10/15s | 480p / 720p / 1080p |
| `seedance-2` | `/uapi/v1/video-gen/seedance` | 5s | 480p / 720p / 1080p |
| `seedance-2-remix` | `/uapi/v1/video-gen/seedance` | 5s | 480p / 720p / 1080p |
| `seedance-2-omni` | `/uapi/v1/video-gen/seedance` | 5s | 480p / 720p / 1080p |
| `kling` | `/uapi/v1/video-gen/kling` | 5/10/15/30s | 480p / 720p / 1080p |

**图片模型：**

| 模型名 | 上游路径 | 分辨率 |
|--------|---------|--------|
| `nano-banana-pro` | `/uapi/v1/generate_image` | 1K / 2K / 4K |
| `nano-banana-2` | `/uapi/v1/generate_image` | 1K / 2K / 4K |
| `imagen-4` | `/uapi/v1/generate_image` | 1K / 2K / 4K |
| `grok-image` | `/uapi/v1/imagen/grok` | - |
| `meta-ai-image` | `/uapi/v1/meta_ai/generate` | - |

---

## 2. 视频生成

### 2.1 提交任务

**请求**

```http
POST /uapi/v1/video-gen/veo
Authorization: Bearer {API_KEY}
Content-Type: multipart/form-data
```

**表单字段：**

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `prompt` | string | ✅ | 视频描述文本（英文效果最佳） |
| `model` | string | ✅ | 视频模型名，如 `veo-3.1`、`grok-3`、`kling` |
| `resolution` | string | ❌ | 分辨率：`480p`（默认）、`720p`、`1080p` |
| `aspect_ratio` | string | ❌ | 宽高比：`16:9`（默认）、`9:16`、`1:1` |
| `seconds` | int | ❌ | 时长（秒）。Veo 默认 8s；Grok 支持 6/10/15；Kling 支持 5/10/15/30 |
| `ref_images` | file / string | ❌ | 参考图片，支持三种格式（与图片生成相同，见上方说明） |
| `mode_image` | string | ❌ | 图片参考模式：`frame`（默认，整图参考）、`ingredient`（局部元素） |

**完整示例（curl）：**

```bash
# Veo 视频
curl -X POST https://heharse.cloud/uapi/v1/video-gen/veo \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A serene sunset over mountains with clouds" \
  -F "model=veo-3.1" \
  -F "resolution=720p" \
  -F "aspect_ratio=16:9"

# Grok 视频
curl -X POST https://heharse.cloud/uapi/v1/video-gen/grok \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A cat playing piano" \
  -F "model=grok-3" \
  -F "seconds=10" \
  -F "resolution=1080p"

# 带参考图（frame 模式）
curl -X POST https://heharse.cloud/uapi/v1/video-gen/veo \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A drone shot flying through mountains" \
  -F "model=veo-3.1" \
  -F "ref_images=@reference.jpg" \
  -F "mode_image=frame"

# Kling 视频（5秒）
curl -X POST https://heharse.cloud/uapi/v1/video-gen/kling \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=Time-lapse of a flower blooming" \
  -F "model=kling" \
  -F "seconds=5" \
  -F "resolution=1080p"
```

**提交成功响应：**

```json
{
  "id": "task_abc123",
  "task_id": "task_abc123",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "object": "video",
  "model": "veo-3.1",
  "status": "queued",
  "progress": 0,
  "created_at": 1713000000
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | New-API 任务 ID（用于后续轮询） |
| `task_id` | string | 同 `id` |
| `uuid` | string | 上游任务 UUID（用于轮询，新版任务才有） |
| `model` | string | 请求的模型名 |
| `status` | string | 状态：`queued`（已提交） |
| `progress` | int | 进度（0-100） |
| `created_at` | int | Unix 时间戳 |

---

### 2.2 轮询任务状态

**请求**

```http
GET /uapi/v1/video-gen/veo?task_id={task_id}
Authorization: Bearer {API_KEY}
```

**说明：** 使用提交响应返回的 `task_id`（推荐），也支持上游的 `uuid`。

**轮询响应（进行中）：**

```json
{
  "code": "success",
  "data": {
    "task_id": "task_abc123",
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "status": "IN_PROGRESS",
    "progress": "45%",
    "data": {
      "type": "video",
      "status": 1,
      "status_desc": "processing",
      "status_percentage": 45
    }
  }
}
```

**轮询响应（成功）：**

```json
{
  "code": "success",
  "data": {
    "task_id": "task_abc123",
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://cdn.example.com/videos/abc123.mp4",
    "data": {
      "type": "video",
      "status": 2,
      "status_desc": "completed",
      "status_percentage": 100,
      "generated_video": [
        {
          "id": 123,
          "uuid": "video-uuid-xxx",
          "video_url": "https://cdn.example.com/videos/abc123.mp4",
          "duration": 8.0,
          "aspect_ratio": "16:9",
          "resolution": "1280x720",
          "status": 2
        }
      ]
    }
  }
}
```

**轮询响应（失败）：**

```json
{
  "code": "success",
  "data": {
    "task_id": "task_abc123",
    "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "status": "FAILURE",
    "progress": "100%",
    "fail_reason": "上游错误信息",
    "data": {
      "type": "video",
      "status": 3,
      "status_desc": "failed",
      "error_code": "ERROR_CODE",
      "error_message": "错误详情"
    }
  }
}
```

**状态码说明：**

| New-API status | 上游 status | 含义 |
|----------------|-------------|------|
| `QUEUED` | 0 | 排队中 |
| `IN_PROGRESS` | 1 | 生成中 |
| `SUCCESS` | 2 | 已完成 |
| `FAILURE` | 3 | 失败 |

---

## 3. 图片生成

### 3.1 提交任务

**请求**

```http
POST /uapi/v1/generate_image
Authorization: Bearer {API_KEY}
Content-Type: multipart/form-data
```

**表单字段：**

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `prompt` | string | ✅ | 图片描述文本 |
| `model` | string | ✅ | 图片模型名，如 `nano-banana-2`、`imagen-4` |
| `resolution` | string | ❌ | 分辨率：`1K`（默认）、`2K`、`4K` |
| `aspect_ratio` | string | ❌ | 宽高比：`1:1`（默认）、`16:9`、`9:16`、`4:3`、`3:4` |
| `style` | string | ❌ | 艺术风格：`Photorealistic`、`Anime General`、`3D Render`、`Illustration` 等 |
| `output_format` | string | ❌ | 输出格式：`jpeg`（默认）、`png` |
| `ref_images` | file / string | ❌ | 参考图片，支持三种格式：<br>1. **multipart 文件**：`ref_images=@file.png`（relay 转为 `files` 上传）<br>2. **base64 data URL**：`ref_images=data:image/png;base64,iVBOR...`（relay 解码后上传）<br>3. **HTTP 图片 URL**：`ref_images=https://example.com/img.png`（relay 转为 `file_urls` 字段，上游下载）<br>单张最大 20MB，最多 3 张 |

**完整示例（curl）：**

```bash
# 基础图片生成
curl -X POST https://heharse.cloud/uapi/v1/generate_image \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=A beautiful landscape with mountains and a lake" \
  -F "model=nano-banana-2" \
  -F "resolution=1K" \
  -F "aspect_ratio=16:9"

# 带参考图
curl -X POST https://heharse.cloud/uapi/v1/generate_image \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=衣服变蓝" \
  -F "model=nano-banana-2" \
  -F "ref_images=@ref.jpg"

# Imagen 4
curl -X POST https://heharse.cloud/uapi/v1/generate_image \
  -H "Authorization: Bearer {API_KEY}" \
  -F "prompt=An astronaut riding a horse" \
  -F "model=imagen-4" \
  -F "resolution=2K"

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

**提交成功响应：**

```json
{
  "id": "task_img_abc123",
  "task_id": "task_img_abc123",
  "uuid": "img-uuid-a1b2c3d4",
  "object": "image",
  "model": "nano-banana-2",
  "status": "queued",
  "progress": 0,
  "created_at": 1713000000
}
```

---

### 3.2 轮询任务状态

**请求**

```http
GET /uapi/v1/generate_image?task_id={task_id}
Authorization: Bearer {API_KEY}
```

**说明：** 使用提交响应返回的 `task_id`（推荐），也支持上游的 `uuid`。

**轮询响应（成功）：**

```json
{
  "code": "success",
  "data": {
    "task_id": "task_img_abc123",
    "uuid": "img-uuid-a1b2c3d4",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://cdn.example.com/images/abc123.jpg",
    "data": {
      "type": "image",
      "status": 2,
      "status_desc": "completed",
      "generated_image": [
        {
          "id": 456,
          "uuid": "gen-img-uuid-xxx",
          "image_url": "https://cdn.example.com/images/abc123.jpg",
          "image_uri": "/images/abc123.jpg",
          "resolution": "1024x1024",
          "model": "nano-banana-2",
          "status": 2
        }
      ],
      "reference_item": [
        {
          "media_type": "image",
          "thumbnail_url": "https://cdn.example.com/thumbs/ref.jpg"
        }
      ]
    }
  }
}
```

---

## 4. 客户端实现示例

### 4.1 Rust（reqwest）

```rust
use reqwest::Client;
use serde::{Deserialize, Serialize};

#[derive(Serialize)]
struct VideoRequest {
    prompt: String,
    model: String,
    resolution: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    seconds: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    ref_images: Option<Vec<u8>>,
}

#[derive(Deserialize)]
struct SubmitResponse {
    task_id: String,
    uuid: Option<String>,
    status: String,
}

async fn submit_video_task(
    client: &Client,
    base_url: &str,
    api_key: &str,
    prompt: &str,
    model: &str,
) -> Result<SubmitResponse, Box<dyn std::error::Error>> {
    let mut form = reqwest::multipart::Form::new()
        .text("prompt", prompt.to_string())
        .text("model", model.to_string())
        .text("resolution", "720p".to_string());

    let resp = client
        .post(format!("{}/uapi/v1/video-gen/veo", base_url))
        .header("Authorization", format!("Bearer {}", api_key))
        .multipart(form)
        .send()
        .await?;

    let body = resp.text().await?;
    let result: SubmitResponse = serde_json::from_str(&body)?;
    Ok(result)
}

async fn poll_task(
    client: &Client,
    base_url: &str,
    api_key: &str,
    task_id: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    loop {
        let resp = client
            .get(format!("{}/uapi/v1/video-gen/veo?task_id={}", base_url, task_id))
            .header("Authorization", format!("Bearer {}", api_key))
            .send()
            .await?;

        let body: serde_json::Value = resp.json().await?;
        let status = body["data"]["status"].as_str().unwrap_or("");

        match status {
            "SUCCESS" => {
                let url = body["data"]["result_url"].as_str().unwrap_or("");
                return Ok(url.to_string());
            }
            "FAILURE" => {
                let reason = body["data"]["fail_reason"].as_str().unwrap_or("unknown");
                return Err(format!("Task failed: {}", reason).into());
            }
            _ => {
                let progress = body["data"]["progress"].as_str().unwrap_or("0%");
                println!("Progress: {}", progress);
                tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
            }
        }
    }
}

// 重要：Windows 上需要禁用代理
let client = Client::builder()
    .no_proxy()  // 解决 Windows 系统代理导致连接失败的问题
    .build()?;
```

### 4.2 Python（requests）

```python
import requests
import time

class GeminiGenAPI:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        self.session.trust_env = False  # 解决 Windows 系统代理问题

    def _headers(self):
        return {'Authorization': f'Bearer {self.api_key}'}

    def submit_video(self, prompt: str, model: str = 'veo-3.1',
                    resolution: str = '720p', seconds: int = None,
                    ref_image: str = None) -> dict:
        """提交视频生成任务"""
        data = {'prompt': prompt, 'model': model, 'resolution': resolution}
        if seconds:
            data['seconds'] = seconds

        files = {}
        if ref_image:
            files['ref_images'] = open(ref_image, 'rb')

        resp = self.session.post(
            f'{self.base_url}/uapi/v1/video-gen/veo',
            headers=self._headers(),
            data=data,
            files=files if files else None
        )
        if files:
            for f in files.values():
                f.close()
        resp.raise_for_status()
        return resp.json()

    def submit_image(self, prompt: str, model: str = 'nano-banana-2',
                    resolution: str = '1K', ref_image: str = None,
                    ref_image_url: str = None) -> dict:
        """提交图片生成任务，ref_image 支持本地文件、base64 data URL、HTTP URL"""
        data = {'prompt': prompt, 'model': model, 'resolution': resolution}
        files = {}
        if ref_image:
            files['ref_images'] = open(ref_image, 'rb')
        elif ref_image_url:
            # HTTP 图片 URL：relay 转为 file_urls 上游下载
            data['ref_images'] = ref_image_url
        # base64 data URL 也可直接放在 ref_images 文本字段，relay 自动解码上传

        resp = self.session.post(
            f'{self.base_url}/uapi/v1/generate_image',
            headers=self._headers(),
            data=data,
            files=files if files else None
        )
        if files:
            for f in files.values():
                f.close()
        resp.raise_for_status()
        return resp.json()

    def poll_video(self, task_id: str, interval: int = 5, timeout: int = 300) -> str:
        """轮询视频任务直到完成，返回视频 URL"""
        start = time.time()
        while time.time() - start < timeout:
            resp = self.session.get(
                f'{self.base_url}/uapi/v1/video-gen/veo?task_id={task_id}',
                headers=self._headers()
            )
            resp.raise_for_status()
            data = resp.json().get('data', {})

            status = data.get('status')
            if status == 'SUCCESS':
                return data.get('result_url', '')
            if status == 'FAILURE':
                raise Exception(f"任务失败: {data.get('fail_reason')}")

            print(f"进度: {data.get('progress', '0%')}")
            time.sleep(interval)

        raise TimeoutError('轮询超时')

    def poll_image(self, task_id: str, interval: int = 5, timeout: int = 120) -> str:
        """轮询图片任务直到完成，返回图片 URL"""
        start = time.time()
        while time.time() - start < timeout:
            resp = self.session.get(
                f'{self.base_url}/uapi/v1/generate_image?task_id={task_id}',
                headers=self._headers()
            )
            resp.raise_for_status()
            data = resp.json().get('data', {})

            status = data.get('status')
            if status == 'SUCCESS':
                return data.get('result_url', '')
            if status == 'FAILURE':
                raise Exception(f"任务失败: {data.get('fail_reason')}")

            print(f"进度: {data.get('progress', '0%')}")
            time.sleep(interval)

        raise TimeoutError('轮询超时')


# 使用示例
api = GeminiGenAPI('https://heharse.cloud', 'YOUR_API_KEY')

# 视频生成
result = api.submit_video('A cat playing piano', model='veo-3.1', resolution='720p')
print(f"任务已提交: {result['task_id']}")

video_url = api.poll_video(result['task_id'])
print(f"视频生成完成: {video_url}")

# 图片生成（带参考图）
result = api.submit_image('衣服变蓝', model='nano-banana-2', ref_image='ref.jpg')
print(f"任务已提交: {result['task_id']}")

image_url = api.poll_image(result['task_id'])
print(f"图片生成完成: {image_url}")
```

### 4.3 JavaScript / TypeScript（fetch）

```typescript
interface TaskResponse {
  task_id: string;
  uuid?: string;
  status: string;
  progress: number;
}

interface PollResponse {
  code: string;
  data: {
    task_id: string;
    uuid?: string;
    status: 'QUEUED' | 'IN_PROGRESS' | 'SUCCESS' | 'FAILURE';
    progress: string;
    result_url?: string;
    fail_reason?: string;
  };
}

class GeminiGenClient {
  constructor(
    private baseURL: string,
    private apiKey: string
  ) {}

  private headers(): HeadersInit {
    return { 'Authorization': `Bearer ${this.apiKey}` };
  }

  async submitVideo(
    prompt: string,
    model: string = 'veo-3.1',
    options: { resolution?: string; seconds?: number; refImage?: File } = {}
  ): Promise<TaskResponse> {
    const formData = new FormData();
    formData.append('prompt', prompt);
    formData.append('model', model);
    if (options.resolution) formData.append('resolution', options.resolution);
    if (options.seconds) formData.append('seconds', String(options.seconds));
    if (options.refImage) formData.append('ref_images', options.refImage);

    const resp = await fetch(`${this.baseURL}/uapi/v1/video-gen/veo`, {
      method: 'POST',
      headers: this.headers(),
      body: formData,
    });
    return resp.json();
  }

  async submitImage(
    prompt: string,
    model: string = 'nano-banana-2',
    options: { resolution?: string; refImage?: File } = {}
  ): Promise<TaskResponse> {
    const formData = new FormData();
    formData.append('prompt', prompt);
    formData.append('model', model);
    if (options.resolution) formData.append('resolution', options.resolution);
    if (options.refImage) formData.append('ref_images', options.refImage);

    const resp = await fetch(`${this.baseURL}/uapi/v1/generate_image`, {
      method: 'POST',
      headers: this.headers(),
      body: formData,
    });
    return resp.json();
  }

  async pollVideo(taskId: string, interval = 5000): Promise<string> {
    while (true) {
      const resp = await fetch(
        `${this.baseURL}/uapi/v1/video-gen/veo?task_id=${taskId}`,
        { headers: this.headers() }
      );
      const result: PollResponse = await resp.json();
      const { status, result_url, fail_reason, progress } = result.data;

      if (status === 'SUCCESS') return result_url!;
      if (status === 'FAILURE') throw new Error(fail_reason);

      console.log(`进度: ${progress}`);
      await new Promise(r => setTimeout(r, interval));
    }
  }

  async pollImage(taskId: string, interval = 5000): Promise<string> {
    while (true) {
      const resp = await fetch(
        `${this.baseURL}/uapi/v1/generate_image?task_id=${taskId}`,
        { headers: this.headers() }
      );
      const result: PollResponse = await resp.json();
      const { status, result_url, fail_reason, progress } = result.data;

      if (status === 'SUCCESS') return result_url!;
      if (status === 'FAILURE') throw new Error(fail_reason);

      console.log(`进度: ${progress}`);
      await new Promise(r => setTimeout(r, interval));
    }
  }
}

// 使用示例
const client = new GeminiGenClient('https://heharse.cloud', 'YOUR_API_KEY');

// 视频生成
const videoTask = await client.submitVideo('A cat playing piano', 'veo-3.1', {
  resolution: '720p',
  seconds: 8
});
console.log(`任务已提交: ${videoTask.task_id}`);
const videoUrl = await client.pollVideo(videoTask.task_id);
console.log(`视频生成完成: ${videoUrl}`);
```

---

## 5. 错误码

| HTTP 状态码 | code | 说明 |
|------------|------|------|
| 401 | - | 认证失败，检查 API Key |
| 429 | - | 请求过于频繁，触发限流 |
| 500 | - | 服务器内部错误 |
| - | `insufficient_user_quota` | 余额不足 |
| - | `model_not_found` | 模型未配置渠道 |
| - | `task_not_exist` | 任务不存在 |
| - | `fail_to_fetch_task` | 上游请求失败（如参考图文件无效） |
| - | `INVALID_FILE_CONTENT` | 参考图文件格式不正确（非图片） |
| - | `INVALID_INPUT` | 请求参数缺失或格式错误 |

**错误响应格式：**

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

## 6. 注意事项

### 6.1 Windows 客户端代理问题

使用 Rust（reqwest）、Python（requests）等客户端在 Windows 上调用时，如遇到连接失败或 `fail_to_fetch_task`，**必须禁用系统代理**：

- **Rust**: `Client::builder().no_proxy().build()`
- **Python**: `session.trust_env = False`
- **JS/TS**: 浏览器环境通常不受影响

### 6.2 参考图字段名

提交时使用 `ref_images` 字段，relay 会自动转换为上游 API 期望的 `files` 字段。

### 6.3 轮询间隔

建议轮询间隔 3-5 秒。New-API 默认轮询间隔为 15 秒。

### 6.4 旧任务兼容

旧版本任务（无 `uuid` 字段）通过 `task_id` 轮询时，New-API 会自动从 `task.Data` 中解析上游 UUID。

---

## 7. 更新历史

| 日期 | 变更内容 |
|------|---------|
| 2026-04-23 | `ref_images` 支持三种格式（文件/base64 data URL/HTTP URL）；`nano-banana-2` 图生图验证通过；文档更新 |
| 2026-04-22 | `/uapi/` 通道修复完成，所有视频/图片接口验证通过；文档完善 |
| 2026-04-22 | 文档初始化，新增所有视频/图片模型接口 |
