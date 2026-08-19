# 灵创 AI（LingchuangAI）渠道对接文档

## 基本信息

- **渠道类型编号**: `71`
- **默认 Base URL**: `https://lingchuangai.top`
- **鉴权方式**: HTTP Bearer Token
  ```
  Authorization: Bearer <API_KEY>
  Content-Type: application/json
  ```

## 支持能力

| 能力 | 接口 | 模式 | 说明 |
|------|------|------|------|
| 视频生成 | `POST /v1/video/generations` | 异步任务 | 提交后返回任务 ID，轮询结果 |
| 图片生成 | `POST /v1/images/generations` | 同步 | 直接返回生成结果 |

## 视频生成

### 提交任务

```bash
curl https://lingchuangai.top/v1/video/generations \
  -H "Authorization: Bearer $LINGCHUANG_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: <唯一键>" \
  -d '{
    "model": "lc-fl-sd2",
    "prompt": "一位女孩在海边奔跑，电影感镜头",
    "duration": 5,
    "aspect_ratio": "16:9",
    "reference_image_urls": ["https://example.com/character.jpg"]
  }'
```

### new-api 字段映射

| new-api 字段 | 灵创 AI 上游字段 | 说明 |
|--------------|------------------|------|
| `model` | `model` | 必填，使用 `GET /v1/models` 返回的 ID |
| `prompt` | `prompt` | 必填，最多 20,000 字符 |
| `duration` / metadata `seconds` | `duration` | 生成时长（秒） |
| `aspect_ratio` / `ratio` / `size` | `aspect_ratio` | 画面比例，默认 `16:9` |
| `reference_images` / `images` / `image_urls` / `image` | `reference_image_urls` | 参考图片 URL 数组 |
| `reference_video` / `video_urls` | `reference_video_urls` | 参考视频 URL 数组 |
| `reference_audio` / `audios` | `reference_audio_urls` | 参考音频 URL 数组 |
| `negative_prompt` | `negative_prompt` | 负面提示词 |
| `generate_audio` | `generate_audio` | 是否生成音频 |
| `watermark` | `watermark` | 是否添加水印 |
| `seed` | `seed` | 随机种子 |
| `resolution` | `resolution` | 分辨率档位 |

### 任务状态

- `queued`: 已接收，等待调度
- `submitting`: 正在提交任务
- `processing`: 任务正在生成或交付
- `succeeded`: 生成成功，`result_url` 可用
- `failed`: 生成失败
- `cancelled`: 任务已取消

## 图片生成

灵创 AI 图片接口为同步返回，new-api 按标准 OpenAI `/v1/images/generations` 流程透传。

```bash
curl https://your-new-api.com/v1/images/generations \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -d '{"model":"lc-image2","prompt":"一张电影感海报","size":"1024x1024","n":1}'
```

## 注意事项

1. **Idempotency-Key**: 视频提交时会自动使用 new-api 生成的公开任务 ID 作为幂等键，重试时保持不变。
2. **素材 URL**: 必须是公开可访问的 HTTPS 直链，不能依赖 Cookie 或登录态。
3. **模型名和价格**: 默认配置了几个示例模型（`lc-fl-sd2`、`lc-image2`），实际模型 ID 和价格请在后台渠道设置中自行配置。
