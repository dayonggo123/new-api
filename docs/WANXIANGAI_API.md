# 万象Ai (WanXiangAI) 渠道对接文档

> **渠道类型**: `万象Ai` (ID: 59)  
> **Base URL**: `https://heharse.cloud`（new-api 对外地址）  
> **认证方式**: `Authorization: Bearer <new-api 的 Token>`

---

## 一、渠道能力概览

万象Ai 渠道支持以下能力：

| 能力 | 类型 | 调用方式 | 说明 |
|------|------|----------|------|
| **对话模型** | Chat | OpenAI 兼容接口 | gpt-5.x / claude-4.x / gemini-3.x 等 |
| **图像生成** | Image | 异步 Task | Nano Banana / GPT Image / 即梦 / Midjourney 等 |
| **视频生成** | Video | 异步 Task | Sora / 可灵 / 万相 / 海螺 / Veo 等 |
| **音频/TTS** | Audio | 异步 Task | speech-2.8 / doubao-tts-2.0 / gemini-tts 等 |
| **音乐生成** | Music | 异步 Task | music-2.5 / music-2.5+ |

> ⚠️ **注意**: 图像/视频/音频/音乐 均为**异步任务**，提交后返回 `task_id`，需轮询查询结果。

---

## 二、对话模型接口

### `POST /v1/chat/completions`

完全兼容 OpenAI Chat Completions API。支持流式输出 (`stream: true`)。

#### 请求参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 模型名称，如 `gpt-5.4`、`claude-sonnet-4-6`、`gemini-3-pro-preview` |
| `messages` | array | ✅ | 消息数组，`[{role: "user", content: "..."}]` |
| `stream` | boolean | ❌ | 是否流式输出，默认 `false` |
| `max_tokens` | integer | ❌ | 最大输出 token 数 |
| `temperature` | float | ❌ | 温度参数，默认由上游决定 |
| `top_p` | float | ❌ | Top-P 采样 |
| `stop` | string/array | ❌ | 停止词 |

#### 请求示例

```bash
curl -X POST https://heharse.cloud/v1/chat/completions \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "你好，请介绍一下自己"}
    ],
    "stream": false,
    "max_tokens": 1024
  }'
```

#### 响应示例（非流式）

```json
{
  "id": "chatcmpl-xxxx",
  "object": "chat.completion",
  "created": 1716200000,
  "model": "gpt-5.4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！我是 GPT-5.4，一个由 OpenAI 训练的大型语言模型..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 25,
    "completion_tokens": 50,
    "total_tokens": 75
  }
}
```

#### 响应示例（流式 SSE）

```
data: {"id":"chatcmpl-xxxx","object":"chat.completion.chunk","choices":[{"delta":{"role":"assistant"}}]}
data: {"id":"chatcmpl-xxxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"你好"}}]}
data: {"id":"chatcmpl-xxxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"！"}}]}
data: [DONE]
```

---

## 三、异步媒体生成接口

图像/视频/音频/音乐 统一使用**异步 Task 接口**提交和查询。

### 3.1 提交生成任务

#### `POST /v1/images/generations` — 图像生成
#### `POST /v1/video/generations` — 视频生成
#### `POST /v1/audio/generations` — 音频/TTS/音乐生成

> 也可统一使用 `POST /v1/videos`（OpenAI 兼容视频接口）提交任意媒体任务。

#### 请求参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 模型名称，如 `wanx2.1-t2i-turbo`、`kling-v3`、`speech-2.8` |
| `prompt` | string | ✅ | 生成描述/文本内容 |
| `image` | string | ❌ | 参考图片 URL（图生图/图生视频时使用） |
| `images` | array | ❌ | 多张参考图片 URL 数组 |
| `size` | string | ❌ | 图像尺寸，如 `1024x1024`、`1792x1024` |
| `duration` | integer | ❌ | 视频/音频时长（秒） |
| `metadata` | object | ❌ | 模型专属扩展参数（见下方说明） |

#### `metadata` 扩展参数

万象Ai 每个模型支持不同的专属参数，通过 `metadata` 对象透传：

```json
{
  "metadata": {
    "aspectRatio": "16:9",
    "voice_id": "your-cloned-voice-id",
    "speed": 1.2,
    "quality": "hd",
    "style": "cinematic"
  }
}
```

常用参数参考：

| 参数名 | 适用类型 | 说明 |
|--------|----------|------|
| `aspectRatio` | 图像/视频 | 画面比例：`1:1`、`16:9`、`9:16`、`3:2`、`2:3` |
| `voice_id` | TTS/音频 | 音色 ID，需先通过万象Ai 克隆 |
| `speed` | TTS | 语速，如 `0.5` ~ `2.0` |
| `lyrics` | 音乐 | 歌词文本（music-2.5 歌曲模式必填） |
| `is_instrumental` | 音乐 | `instrumental` 表示纯音乐，不传则走歌曲模式 |

#### 请求示例 — 文生图

```bash
curl -X POST https://heharse.cloud/v1/images/generations \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "wanx2.1-t2i-turbo",
    "prompt": "一只戴着墨镜的猫在海滩上冲浪",
    "size": "1024x1024",
    "metadata": {"aspectRatio": "1:1"}
  }'
```

#### 请求示例 — 视频生成

```bash
curl -X POST https://heharse.cloud/v1/video/generations \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v3-video",
    "prompt": "一只猫咪在草地上奔跑",
    "duration": 5,
    "metadata": {"aspectRatio": "16:9", "quality": "hd"}
  }'
```

#### 请求示例 — TTS 语音合成

```bash
curl -X POST https://heharse.cloud/v1/audio/generations \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8",
    "prompt": "你好，这是测试语音。今天天气真不错。",
    "metadata": {
      "voice_id": "your-cloned-voice-id",
      "speed": 1.0,
      "quality": "HD"
    }
  }'
```

#### 请求示例 — 音乐生成

```bash
curl -X POST https://heharse.cloud/v1/audio/generations \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.5",
    "prompt": "一首关于夏日海滩的流行歌曲",
    "metadata": {
      "lyrics": "阳光洒在沙滩上，海浪轻轻拍打岸边...",
      "style": "pop",
      "structure": "verse-chorus"
    }
  }'
```

#### 响应示例（提交成功）

```json
{
  "id": "task-xxxxxx",
  "object": "video",
  "model": "kling-v3-video",
  "status": "queued",
  "created_at": 1716200000,
  "task_id": "task-xxxxxx"
}
```

---

### 3.2 查询任务状态

#### `GET /v1/videos/:task_id` — 通用视频/音频任务查询
#### `GET /v1/audio/generations/:task_id` — 音频任务查询
#### `GET /v1/video/generations/:task_id` — 视频任务查询

> 以上查询接口互通，任意 `task_id` 可通过任一接口查询。

#### 响应示例 — 任务进行中

```json
{
  "id": "task-xxxxxx",
  "object": "video",
  "model": "kling-v3-video",
  "status": "in-progress",
  "progress": 45,
  "created_at": 1716200000
}
```

#### 响应示例 — 任务完成

```json
{
  "id": "task-xxxxxx",
  "object": "video",
  "model": "kling-v3-video",
  "status": "completed",
  "progress": 100,
  "created_at": 1716200000,
  "completed_at": 1716200120,
  "metadata": {
    "url": "https://generated-media-cdn.example.com/result.mp4",
    "type": "video"
  }
}
```

#### 响应示例 — 任务失败

```json
{
  "id": "task-xxxxxx",
  "object": "video",
  "model": "kling-v3-video",
  "status": "failed",
  "created_at": 1716200000,
  "error": {
    "message": "生成失败：提示词包含敏感内容",
    "code": "task_failed"
  }
}
```

#### 状态值说明

| 状态值 | 含义 | 是否终态 |
|--------|------|----------|
| `queued` | 排队中 | ❌ |
| `in-progress` | 生成中 | ❌ |
| `completed` | 已完成 | ✅ |
| `failed` | 失败 | ✅ |

#### 轮询建议

```
提交任务 → 等待 5 秒 → 首次轮询
         → 每 5 秒轮询一次
         → status 为 completed/failed 时停止
```

> ⚠️ 部分模型（如视频）上游不返回中间进度，`progress` 可能全程为 `0`，完成时瞬间跳到 `100`。不要因此提前判定为失败。

---

## 四、模型名称参考

以下是在 new-api 中配置渠道时使用的**模型名称**（`model` 字段传的值）：

### 对话模型 (Chat)

| 模型名称 | 类型 | 说明 |
|----------|------|------|
| `gpt-5.4` | OpenAI | GPT-5.4 旗舰 |
| `gpt-5.5` | OpenAI | GPT-5.5 |
| `gpt-5.5-xhigh` | OpenAI | GPT-5.5 深度推理 |
| `gpt-5.5-high` | OpenAI | GPT-5.5 高推理 |
| `claude-opus-4-7` | Anthropic | Claude Opus 4.7 |
| `claude-sonnet-4-6` | Anthropic | Claude Sonnet 4.6 |
| `claude-haiku-4-5-20251001` | Anthropic | Claude Haiku 4.5 |
| `gemini-3-pro-preview` | Gemini | Gemini 3 Pro |
| `gemini-3.1-pro-preview` | Gemini | Gemini 3.1 Pro |
| `gemini-3.1-flash-lite-preview` | Gemini | Gemini 3.1 Flash |

### 图像模型 (Image)

| 模型名称 | 说明 |
|----------|------|
| `wanx2.1-t2i-turbo` | 万相 2.1 文生图 |
| `gpt-image-2` | GPT Image 2 |
| `mj_imagine` | Midjourney |
| `doubao-seedream-5-0-260128` | 即梦 5.0 |
| `grok-4.2-image` | Grok 4.2 图像 |
| `kling-v3` | 可灵 V3 |
| `kling-v3-omni` | 可灵 V3 Omni |

### 视频模型 (Video)

| 模型名称 | 说明 |
|----------|------|
| `sora-2` | Sora-2 官转 |
| `kling-v3-video` | 可灵 V3 视频 |
| `kling-v2-6` | 可灵 2.6 Pro |
| `wan2.7-shouweizhen` | 万相 2.7 首尾帧 |
| `wan2.7-cankaosheng` | 万相 2.7 参考生 |
| `veo3.1` | Veo 3.1 |
| `hailuo-2.3` | 海螺 2.3 |
| `viduq3` | Vidu Q3 |

### 音频/TTS/音乐模型 (Audio)

| 模型名称 | 类型 | 说明 |
|----------|------|------|
| `speech-2.8` | TTS | 海螺语音克隆 2.8 |
| `doubao-tts-2.0` | TTS | 豆包语音合成 2.0 |
| `gemini-2.5-pro-preview-tts` | TTS | Gemini 2.5 TTS |
| `gemini-3.1-flash-tts-preview` | TTS | Gemini 3.1 Flash TTS |
| `music-2.5` | 音乐 | 海螺音乐生成 |
| `music-2.5+` | 音乐 | 海螺音乐生成增强版 |

---

## 五、错误码

### HTTP 状态码

| 状态码 | 含义 |
|--------|------|
| `200` | 成功 |
| `400` | 请求参数错误 |
| `401` | Token 无效或过期 |
| `402` | 余额不足 |
| `429` | 请求过于频繁 |
| `500` | 上游服务错误 |

### 常见错误响应

```json
{
  "error": {
    "message": "模型 gpt-5.4 在当前分组下无可用渠道",
    "type": "new_api_error",
    "code": "model_not_found"
  }
}
```

---

## 六、渠道配置说明

在 new-api 管理后台添加万象Ai渠道时：

| 配置项 | 值 |
|--------|-----|
| **渠道类型** | `万象Ai` (59) |
| **名称** | 自定义，如 "万象Ai-对话" |
| **Base URL** | `https://api.lk888.ai`（或留空使用默认） |
| **密钥** | 万象Ai 平台的 API Key (`sk-...`) |
| **模型** | 手动填写，如 `gpt-5.4`、`wanx2.1-t2i-turbo`、`speech-2.8` |

> 万象Ai **不支持**标准 `/v1/models` 自动获取，模型列表需手动维护。
