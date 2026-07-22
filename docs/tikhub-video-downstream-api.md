# TikHub 单视频 V2 下游对接文档

## 接口说明

new-api 代理 TikHub `/api/v1/tiktok/app/v3/fetch_one_video_v2` 接口，下游通过 new-api 的 Token 鉴权调用，new-api 再使用后台配置的 TikHub API Key 向上游转发。

> **注意**: TikHub API Base URL 已更新为 `https://heharse.cloud`

## 接口地址

```http
GET /api/public/tikhub/tiktok/video
```

完整 URL 示例：

```text
https://你的域名/api/public/tikhub/tiktok/video?aweme_id=7350810998023949599
```

## 鉴权

在请求头中携带 new-api 的 API Key：

```http
Authorization: Bearer <new-api API Key>
```

API Key 在 new-api 后台「令牌」页面创建，权限需开启调用权限。

## 请求参数

### Query 参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `aweme_id` | string | 是 | TikTok 作品 ID，如 `7350810998023949599` |

### 请求示例

```bash
curl -X GET "https://你的域名/api/public/tikhub/tiktok/video?aweme_id=7350810998023949599" \
  -H "Authorization: Bearer sk-xxxx"
```

## 响应说明

new-api 直接透传 TikHub 上游返回的原始 JSON，HTTP 状态码保持上游一致（通常为 200）。

### 成功响应示例（200）

```json
{
  "code": 200,
  "request_id": "xxx",
  "message": "Request successful. This request will incur a charge.",
  "message_zh": "请求成功，本次请求将被计费。",
  "support": "Discord: https://discord.gg/aMEAS8Xsvz",
  "time": "2026-07-10 00:00:00",
  "time_stamp": 1752096000,
  "time_zone": "America/Los_Angeles",
  "docs": null,
  "cache_message": null,
  "cache_message_zh": null,
  "cache_url": null,
  "router": "",
  "params": "",
  "data": {
    "aweme_id": "7350810998023949599",
    "title": "...",
    "desc": "...",
    "author": { ... },
    "statistics": { ... }
  }
}
```

### 错误响应示例

#### 参数缺失（400）

```json
{
  "success": false,
  "message": "aweme_id 不能为空"
}
```

#### 接口未启用（503）

```json
{
  "success": false,
  "message": "TikHub 接口未启用"
}
```

#### 鉴权失败（401）

由 new-api 统一鉴权中间件返回，常见原因：

- API Key 不存在或已禁用
- API Key 无调用权限

```json
{
  "success": false,
  "message": "Unauthorized"
}
```

#### 上游异常（502）

new-api 转发到 TikHub 失败时返回：

```json
{
  "success": false,
  "message": "tikhub api returned status 401: {上游返回内容}"
}
```

## 多语言示例

### Python

```python
import requests

url = "https://你的域名/api/public/tikhub/tiktok/video"
headers = {"Authorization": "Bearer sk-xxxx"}
params = {"aweme_id": "7350810998023949599"}

resp = requests.get(url, headers=headers, params=params, timeout=30)
print(resp.status_code)
print(resp.json())
```

### JavaScript / Node.js

```javascript
const response = await fetch(
  'https://你的域名/api/public/tikhub/tiktok/video?aweme_id=7350810998023949599',
  {
    headers: { 'Authorization': 'Bearer sk-xxxx' }
  }
);
const data = await response.json();
console.log(data);
```

### Go

```go
package main

import (
    "fmt"
    "io"
    "net/http"
)

func main() {
    req, _ := http.NewRequest("GET", "https://你的域名/api/public/tikhub/tiktok/video?aweme_id=7350810998023949599", nil)
    req.Header.Set("Authorization", "Bearer sk-xxxx")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))
}
```

## 注意事项

1. **鉴权隔离**：下游只需要 new-api 的 API Key，不需要知道 TikHub 的 API Key。
2. **计费**：当前版本仅做代理转发，不扣除 new-api 额度。如需按次计费，需后续扩展。
3. **超时**：new-api 向上游请求超时为 30 秒，若 TikHub 响应慢可能返回 502。
4. **缓存**：当前版本未做缓存，每次请求都会转发到 TikHub。如需要缓存，可后续根据 TikHub 响应中的 `cache_url` 或业务场景增加。
5. **aweme_id 格式**：必须是纯数字字符串，不要带多余空格或前缀。

## 后台启用方式

1. 管理员登录 new-api 后台。
2. 进入「运营设置」→「TikHub 设置」。
3. 开启「启用 TikHub 接口代理」，填写 TikHub API Key。
4. 保存后下游即可调用。

## 相关文件

- 后端接口实现：`controller/tikhub.go`
- 上游转发逻辑：`service/tikhub.go`
- 配置定义：`setting/operation_setting/tikhub_setting.go`
- 路由注册：`router/api-router.go`
- 后台配置页面：`web/src/pages/Setting/Operation/SettingsTikHub.jsx`
