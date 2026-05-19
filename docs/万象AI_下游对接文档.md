# 万象AI / GeminiGen 下游应用对接文档

## 基础信息

| 项目 | 值 |
|------|-----|
| 网关地址 | `https://heharse.cloud` |
| 认证方式 | `Authorization: Bearer sk-xxx` |

## 支持模型

- `nano-banana-pro`
- `nano-banana-2`
- `veo-3.1-fast`
- `veo-3.1-lite`
- `veo-3.1-4k`
- `gemini-3-pro`

> 渠道配置：GeminiGen（优先级10，主渠道）→ 万象AI（优先级5，备用渠道）

---

## 接口一：标准聊天补全（/v1/chat/completions）

**适用场景**：下游应用只支持标准 OpenAI 聊天接口（如 ChatGPT-Next-Web、LobeChat 等）

**特点**：
- ✅ 兼容性最好，无需改代码
- ❌ 不创建任务日志
- ❌ 不轮询任务状态

### 请求

```http
POST /v1/chat/completions
Authorization: Bearer sk-xxx
Content-Type: application/json
```

```json
{
  "model": "nano-banana-pro",
  "messages": [
    {"role": "user", "content": "a beautiful sunset over ocean"}
  ]
}
```

### 响应

```json
{
  "id": "video-xxx",
  "object": "video",
  "status": "completed",
  "model": "nano-banana-pro",
  "url": "https://cdn.xxx.com/result.png"
}
```

---

## 接口二：媒体生成任务（/v1/media/generate）⭐ 推荐

**适用场景**：自研应用，需要任务日志和异步轮询

**特点**：
- ✅ 创建任务记录
- ✅ 支持任务状态轮询
- ✅ 支持计费
- ✅ 主备渠道自动切换

### 1. 提交任务

```http
POST /v1/media/generate
Authorization: Bearer sk-xxx
Content-Type: application/json
```

```json
{
  "model": "nano-banana-pro",
  "prompt": "a beautiful sunset over ocean",
  "size": "1024x1024"
}
```

### 响应

```json
{
  "id": "task_xxxxxx",
  "object": "video",
  "status": "queued",
  "model": "nano-banana-pro",
  "created_at": 1715932800
}
```

### 2. 轮询查询任务状态

```http
GET /v1/media/generate/{task_id}
Authorization: Bearer sk-xxx
```

**完成响应**：

```json
{
  "id": "task_xxxxxx",
  "object": "video",
  "status": "completed",
  "model": "nano-banana-pro",
  "url": "https://cdn.xxx.com/result.png",
  "created_at": 1715932800,
  "completed_at": 1715932900
}
```

**失败响应**：

```json
{
  "id": "task_xxxxxx",
  "object": "video",
  "status": "failed",
  "model": "nano-banana-pro",
  "error": {
    "message": "generation failed",
    "code": "task_failed"
  }
}
```

---

## 接口三：图片生成（/v1/images/generations）

**适用场景**：下游应用支持 DALL-E 图片生成接口

### 请求

```http
POST /v1/images/generations
Authorization: Bearer sk-xxx
Content-Type: application/json
```

```json
{
  "model": "nano-banana-pro",
  "prompt": "a beautiful sunset over ocean",
  "n": 1,
  "size": "1024x1024"
}
```

### 响应

```json
{
  "created": 1715932800,
  "data": [
    {
      "url": "https://cdn.xxx.com/result.png"
    }
  ]
}
```

---

## 下游客户端配置指南

### ChatGPT-Next-Web / LobeChat / ChatBox

1. 接口地址：`https://heharse.cloud`
2. API Key：你的令牌
3. 自定义模型：添加 `nano-banana-pro,nano-banana-2,veo-3.1-fast,veo-3.1-lite,veo-3.1-4k,gemini-3-pro`
4. 选择模型后正常聊天即可

### 自研应用（推荐走 /v1/media/generate）

```python
import requests
import time

BASE_URL = "https://heharse.cloud"
API_KEY = "sk-xxx"

# 1. 提交任务
resp = requests.post(
    f"{BASE_URL}/v1/media/generate",
    headers={"Authorization": f"Bearer {API_KEY}"},
    json={
        "model": "nano-banana-pro",
        "prompt": "a beautiful sunset over ocean",
        "size": "1024x1024"
    }
)
task = resp.json()
task_id = task["id"]

# 2. 轮询查询
while True:
    resp = requests.get(
        f"{BASE_URL}/v1/media/generate/{task_id}",
        headers={"Authorization": f"Bearer {API_KEY}"}
    )
    result = resp.json()
    if result["status"] in ("completed", "failed"):
        break
    time.sleep(2)

print(result)
```

---

## 主备切换说明

当前配置：
- **GeminiGen**（优先级10）→ 主渠道
- **万象AI**（优先级5）→ 备用渠道

系统行为：
1. 第1次请求 → GeminiGen
2. GeminiGen 失败 → 自动重试 → 万象AI
3. 两个渠道模型名完全一致，通过模型重定向调用各自上游
