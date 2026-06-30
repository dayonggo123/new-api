# APIMart Whisper-1 渠道配置说明

## 概述

new-api 已内置 **APIMart** 渠道类型（Channel Type = 61）。APIMart 的 Whisper-1 音频转录接口遵循 OpenAI 兼容格式，因此可以直接通过 new-api 标准 relay 调用，无需额外代理控制器。

## 配置步骤

### 1. 添加渠道

进入管理后台 → **渠道 → 添加渠道**：

| 字段 | 值 |
|------|-----|
| 渠道类型 | **APIMart** |
| 名称 | APIMart Whisper-1（自定义） |
| Base URL | `https://api.apimart.ai` |
| 密钥 | 从 [APIMart](https://apimart.ai/keys) 获取的 API Key |
| 模型 | 添加 `whisper-1` |

### 2. 配置模型倍率（如需计费）

`whisper-1` 已在 `setting/ratio_setting/model_ratio.go` 中预置默认倍率，通常无需额外操作。

### 3. 下游调用

下游应用使用 new-api 的标准 OpenAI 兼容接口：

```bash
POST https://你的new-api域名/v1/audio/transcriptions
Authorization: Bearer sk-xxxxxx
Content-Type: multipart/form-data
```

参数与 APIMart / OpenAI 完全一致：

| 参数 | 说明 |
|------|------|
| `file` | 音频文件（mp3/mp4/mpeg/mpga/m4a/wav/webm），最大 25MB |
| `model` | 固定 `whisper-1` |
| `language` | 可选，语言代码如 `zh`、`en` |
| `prompt` | 可选，提示文本 |
| `response_format` | 可选：`json`（默认）/`text`/`srt`/`verbose_json`/`vtt` |
| `temperature` | 可选，0 ~ 1 |

### 4. 请求示例

```bash
curl --request POST \
  --url 'https://你的new-api域名/v1/audio/transcriptions' \
  --header 'Authorization: Bearer sk-xxxxxx' \
  --form 'file=@/path/to/audio.mp3' \
  --form 'model=whisper-1' \
  --form 'language=zh' \
  --form 'response_format=json'
```

## 响应示例

```json
{
  "text": "这是一段测试音频的转录文本内容。"
}
```

## 注意事项

1. APIMart 渠道同时支持 task 图像模型（如 `gpt-image-2`）和 OpenAI 兼容接口（如 `whisper-1`）。
2. 图像任务会在 `image_handler` 中按渠道类型直接拦截走异步 task 流程，不影响 Whisper 等标准接口。
3. 如果只需要 Whisper，可专门创建一个只绑定 `whisper-1` 模型的 APIMart 渠道。
4. 所有计费、限流、重试、日志均走 new-api 标准渠道逻辑。
