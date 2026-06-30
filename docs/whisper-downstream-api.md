# Whisper-1 音频转录接口 — 下游调用文档

## 接口地址

```
POST https://你的new-api域名/api/public/audio/transcriptions
```

## 鉴权方式

在请求头中传入 new-api 的 API Key：

```
Authorization: Bearer sk-xxxxxx
```

## 请求参数

请求使用 `multipart/form-data` 格式，参数原样透传给 APIMart Whisper-1。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file` | file | 是 | 音频文件，支持 mp3、mp4、mpeg、mpga、m4a、wav、webm，最大 25 MB |
| `model` | string | 是 | 固定为 `whisper-1` |
| `language` | string | 否 | 音频语言代码（ISO-639-1），如 `zh`、`en`、`ja`、`ko` |
| `prompt` | string | 否 | 可选文本提示，用于指导转录风格，最长 224 tokens |
| `response_format` | string | 否 | 输出格式：`json`（默认）、`text`、`srt`、`verbose_json`、`vtt` |
| `temperature` | number | 否 | 采样温度，范围 `0` ~ `1`，默认 `0` |

## 请求示例

### cURL

```bash
curl --request POST \
  --url 'https://你的new-api域名/api/public/audio/transcriptions' \
  --header 'Authorization: Bearer sk-xxxxxx' \
  --header 'Content-Type: multipart/form-data' \
  --form 'file=@/path/to/audio.mp3' \
  --form 'model=whisper-1' \
  --form 'language=zh' \
  --form 'response_format=json'
```

### Python

```python
import requests

url = 'https://你的new-api域名/api/public/audio/transcriptions'
files = {'file': open('/path/to/audio.mp3', 'rb')}
data = {
    'model': 'whisper-1',
    'language': 'zh',
    'response_format': 'json',
}
headers = {'Authorization': 'Bearer sk-xxxxxx'}

response = requests.post(url, files=files, data=data, headers=headers)
print(response.json())
```

### JavaScript

```javascript
const formData = new FormData();
formData.append('file', audioFile);
formData.append('model', 'whisper-1');
formData.append('language', 'zh');
formData.append('response_format', 'json');

fetch('https://你的new-api域名/api/public/audio/transcriptions', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer sk-xxxxxx',
  },
  body: formData,
})
  .then(response => response.json())
  .then(data => console.log(data))
  .catch(error => console.error('Error:', error));
```

## 响应示例

### `response_format=json`（默认）

```json
{
  "text": "这是一段测试音频的转录文本内容。"
}
```

### `response_format=verbose_json`

```json
{
  "task": "transcribe",
  "language": "zh",
  "duration": 8.5,
  "text": "这是一段测试音频的转录文本内容。",
  "segments": [
    {
      "id": 0,
      "seek": 0,
      "start": 0.0,
      "end": 3.5,
      "text": "这是一段测试音频",
      "tokens": [50364, 1234, 5678],
      "temperature": 0.0,
      "avg_logprob": -0.3,
      "compression_ratio": 1.2,
      "no_speech_prob": 0.01
    }
  ]
}
```

### `response_format=srt`

```srt
1
00:00:00,000 --> 00:00:03,500
这是一段测试音频

2
00:00:03,500 --> 00:00:08,500
的转录文本内容。
```

## 错误响应

```json
{
  "success": false,
  "message": "Whisper 接口未启用"
}
```

```json
{
  "success": false,
  "message": "missing file field"
}
```

## 注意事项

1. 请求必须使用 `multipart/form-data` 格式
2. 音频文件大小不能超过 25 MB
3. 所有请求必须携带有效的 `Authorization` 头，否则返回 `401` 未授权
4. new-api 会将请求原样透传给 APIMart，响应也直接透传
5. 建议指定 `language` 以提高识别准确率和速度
