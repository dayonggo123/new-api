# PRD：new-api 接入火山方舟/豆包视频生成能力

## 1. 项目信息

| 字段 | 值 |
|---|---|
| Language | 中文 |
| Programming Language | Go（new-api 后端） |
| Project Name | new_api_volcengine_video_generation |
| 原始需求 | 在 new-api 中接入火山方舟（VolcEngine）及豆包通用/豆包视频渠道的视频生成能力，支持文生视频、图生视频、多模态参考视频生成，复用现有 task 机制实现异步任务+轮询。 |

## 2. 产品定义

### 2.1 产品目标（一句话）

让 new-api 用户通过统一的 OpenAI 兼容接口 `/v1/videos/generations` 调用火山方舟/豆包的视频生成模型，复用现有 task 平台实现异步创建、轮询、结果回传，补齐 new-api 在多模态视频生成领域的渠道覆盖能力。

### 2.2 用户故事

- **作为** new-api 平台运营者，**我希望**在渠道配置中新增“火山方舟-视频生成”或“豆包视频”渠道，**从而**让下游用户无需关心官方 API 差异即可发起视频生成任务。
- **作为** 使用 OpenAI 兼容 SDK 的开发者，**我希望**调用 `/v1/videos/generations` 并传入 prompt/图片/参考视频，**从而**以与图像生成一致的体验获得生成视频 URL。
- **作为** new-api 管理员，**我希望**视频生成任务进入现有 task 平台进行状态轮询和生命周期管理，**从而**复用现有监控、计费和重试机制，降低运维成本。

## 3. 需求池

### P0（Must have，必须实现）

1. **新增渠道类型映射**：在 new-api 渠道配置中识别 VolcEngine 视频生成与 DoubaoVideo 渠道，路由到正确的 task adaptor。
2. **OpenAI 兼容入口**：实现 `/v1/videos/generations` 接口，接收 `model`、`prompt`、`image`（可选）、`size`/`ratio`、`duration`、`resolution` 等字段，向下转发为火山方舟创建任务请求。
3. **创建任务**：调用 `POST /api/v3/contents/generations/tasks`，将模型 ID、content 数组、generate_audio、ratio、duration、resolution 等字段正确映射，返回 task id 给 task 平台。
4. **查询任务**：调用 `GET /api/v3/contents/generations/tasks/{id}`，解析 `status`（queued/running/succeeded/failed/expired/cancelled）及 `content.video_url`、`content.last_frame_url`、`usage.completion_tokens`，完成 task 状态流转。
5. **结果回传**：当任务成功时，将官方响应转换为 OpenAI 兼容的视频生成结果格式（包含视频 URL、封面/最后一帧 URL、usage 信息），通过 SSE 或轮询接口返回给客户端。

### P1（Should have，建议实现）

6. **多模态参考支持**：支持 `first_frame`、`last_frame`、`reference_image`、`reference_video`、`reference_audio` 等 role 的 content 映射，允许用户上传参考图/视频/音频作为生成条件。
7. **参数透传**：支持 `seed`、`watermark`、`camera_fixed`、`service_tier`、`priority`、`frames` 等高级参数透传；`size` 字段支持 OpenAI 常见格式（如 `1920x1080`）到 `ratio` 的智能映射。
8. **错误处理与兜底**：对 failed/expired/cancelled 状态提供统一错误码和可读错误信息；对网络超时、API Key 无效、模型不可用等场景返回标准 HTTP 状态码。

### P2（Nice to have，可延后）

9. **模型系列自动识别**：在渠道模型列表中预置 Seedance 2.0 / 1.5 Pro / 1.0 Pro / 1.0 Pro Fast 等模型别名，支持用户配置真实模型 ID 与显示名称的映射。
10. **回调优化**：支持 `callback_url` 透传，允许用户配置异步回调，减少 task 轮询频率。
11. **视频结果缓存/转存**：对生成的视频 URL 进行短期缓存或转存到对象存储，避免官方 URL 过期导致客户端无法下载。

## 4. 接口映射：客户端 OpenAI 兼容请求 → 火山方舟官方接口

### 4.1 客户端请求

```http
POST /v1/videos/generations
Content-Type: application/json
Authorization: Bearer {new-api-key}

{
  "model": "doubao-seedance-2-0-260128",
  "prompt": "一只橘猫在夕阳下的屋顶上奔跑，电影感镜头",
  "image": "https://example.com/cat.jpg",
  "size": "16:9",
  "duration": 5,
  "resolution": "720p",
  "generate_audio": true
}
```

### 4.2 映射到火山方舟创建任务

```http
POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks
Authorization: Bearer {volcengine-api-key}
Content-Type: application/json

{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {
      "type": "text",
      "role": "user",
      "content": "一只橘猫在夕阳下的屋顶上奔跑，电影感镜头"
    },
    {
      "type": "image_url",
      "role": "first_frame",
      "image_url": "https://example.com/cat.jpg"
    }
  ],
  "generate_audio": true,
  "ratio": "16:9",
  "duration": 5,
  "resolution": "720p",
  "watermark": false
}
```

### 4.3 字段映射说明

| 客户端字段（OpenAI 兼容） | 火山方舟字段 | 映射规则 |
|---|---|---|
| `model` | `model` | 直接透传真实模型 ID。 |
| `prompt` | `content[].content`（type=text, role=user） | 必填，作为文本提示词。 |
| `image` / `images` | `content[].image_url`（type=image_url, role=first_frame/reference_image） | 单图默认映射为 `first_frame`；如显式指定 `reference_image` 则按用户参数处理。 |
| `video` / `reference_video` | `content[].video_url`（type=video_url, role=reference_video） | 多模态参考视频，支持 URL 或 base64。 |
| `audio` / `reference_audio` | `content[].audio_url`（type=audio_url, role=reference_audio） | 多模态参考音频。 |
| `size` / `ratio` | `ratio` | 优先使用 `ratio`；`size` 按常见比例映射（如 1920x1080→16:9，1080x1920→9:16，512x512→1:1）。 |
| `duration` | `duration` | 直接透传，默认 5。 |
| `resolution` | `resolution` | 直接透传，默认 720p。 |
| `generate_audio` | `generate_audio` | 直接透传，默认 true。 |
| `seed` | `seed` | 直接透传。 |
| `watermark` | `watermark` | 直接透传，默认 false。 |
| `camera_fixed` / `frames` / `priority` 等 | 同名字段 | 直接透传。 |
| `callback_url` | `callback_url` | 可选透传，P2 考虑。 |

### 4.4 查询任务与结果回传

火山方舟响应：

```json
{
  "id": "task-xxx",
  "status": "succeeded",
  "content": {
    "video_url": "https://cdn.example.com/output.mp4",
    "last_frame_url": "https://cdn.example.com/last_frame.jpg"
  },
  "usage": {
    "completion_tokens": 1
  }
}
```

new-api 返回客户端（OpenAI 兼容）：

```json
{
  "created": 1718000000,
  "data": [
    {
      "url": "https://cdn.example.com/output.mp4",
      "last_frame_url": "https://cdn.example.com/last_frame.jpg",
      "revised_prompt": "一只橘猫在夕阳下的屋顶上奔跑，电影感镜头"
    }
  ],
  "usage": {
    "completion_tokens": 1
  }
}
```

## 5. 待确认问题

1. **模型 ID 与渠道路由**：
   - 火山方舟渠道与豆包视频渠道（DoubaoVideo）在 new-api 中的路由规则是否完全复用现有 `taskdoubao` 适配器？是否需要新增一个独立的 `volcengine_video` adaptor 文件，还是直接扩展 `relay/channel/task/doubao`？
   - 真实模型 ID 是否需要通过渠道配置中的模型映射表来管理，还是直接使用用户传入的 `model` 字段？

2. **多模态参考字段的 OpenAI 兼容表达**：
   - 对于 `first_frame` / `last_frame` / `reference_image` / `reference_video` / `reference_audio` 这些 role，客户端请求是否统一使用 `image`、`reference_image`、`reference_video`、`reference_audio` 字段，还是采用 content 数组格式（类似 GPT-4o 的多模态 messages）？
   - 参考资源是否支持 base64，还是仅支持 URL？new-api 是否负责上传资源到临时存储？

3. **异步结果返回机制**：
   - 视频生成任务通常耗时数秒到数分钟，new-api 是否保持现有 task 平台的轮询间隔（如 1s/2s/5s），还是需要针对视频生成任务设置更长的默认轮询周期和超时时间？
   - 客户端是否需要新增 `/v1/videos/generations/{task_id}` 或 `/v1/tasks/{id}` 查询接口，还是仅通过 task 平台内部 SSE 返回？

4. **计费与用量统计**：
   - 火山方舟返回的 `usage.completion_tokens` 是否足以支撑计费，还是需要按视频时长、分辨率等维度进行额外计价？
   - 是否需要在模型价格表中新增视频生成模型的计费单位（如 per-video / per-second）？

5. **错误状态与重试策略**：
   - 当官方返回 `failed` 或 `expired` 时，new-api 是否自动重试创建任务，还是仅返回错误给客户端？
   - 任务取消（`cancelled`）是否由 new-api 暴露取消接口，还是仅依赖官方状态同步？
