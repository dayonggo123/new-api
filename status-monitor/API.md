# GeminiGen Status API 对接文档

> 自动抓取 https://geminigen.ai/status/ 的实时状态数据，提供 JSON API 供第三方对接。

---

## 服务信息

| 项目 | 值 |
|------|-----|
| **服务地址** | `http://你的IP:3456` |
| **默认端口** | `3456`（可通过环境变量 `STATUS_PORT` 修改） |
| **抓取间隔** | 每 60 秒自动抓取一次 |
| **数据来源** | https://geminigen.ai/status/ |
| **CORS** | 默认允许所有域名跨域访问 |

---

## 接口列表

### 1. 获取状态数据

```
GET /api/status
```

**响应示例：**

```json
{
  "fetchedAt": "2026-05-18T15:09:20.244Z",
  "source": "https://geminigen.ai/status/",
  "total": 21,
  "models": [
    {
      "category": "video",
      "categoryRaw": "TẠO VIDEO",
      "name": "Veo 3.1",
      "status": "Sự cố gián đoạn một phần",
      "statusCode": "partial_outage",
      "successRate": 43.6
    },
    {
      "category": "video",
      "categoryRaw": "TẠO VIDEO",
      "name": "Veo 2",
      "status": "Hoạt động",
      "statusCode": "operational",
      "successRate": 100
    },
    {
      "category": "image",
      "categoryRaw": "TẠO ẢNH",
      "name": "Imagen 4",
      "status": "Sự cố gián đoạn một phần",
      "statusCode": "partial_outage",
      "successRate": 47.0
    }
  ]
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `fetchedAt` | string (ISO 8601) | 数据抓取时间 |
| `source` | string | 数据源地址 |
| `total` | number | 模型总数 |
| `models` | array | 模型列表 |
| `models[].category` | string | 分类：`video` / `image` |
| `models[].categoryRaw` | string | 原始分类名（越南语） |
| `models[].name` | string | 模型名称 |
| `models[].status` | string | 原始状态文本（越南语） |
| `models[].statusCode` | string | 状态码：`operational` / `degraded` / `partial_outage` / `unknown` |
| `models[].successRate` | number | 成功率（百分比） |

**statusCode 对照表：**

| statusCode | 含义 | 建议颜色 |
|-----------|------|---------|
| `operational` | 正常运行 | 🟢 绿色 |
| `degraded` | 性能下降 | 🟡 黄色 |
| `partial_outage` | 部分中断 | 🟠 橙色 |
| `unknown` | 未知状态 | ⚪ 灰色 |

---

### 2. 健康检查

```
GET /health
```

**响应示例：**

```json
{
  "status": "ok",
  "lastFetch": "2026-05-18T15:09:20.244Z",
  "modelCount": 21
}
```

用于监控服务本身是否存活、上次抓取时间、模型数量。

---

## 对接示例

### cURL

```bash
curl http://localhost:3456/api/status
```

### JavaScript / fetch

```javascript
async function getStatus() {
  const res = await fetch('http://localhost:3456/api/status');
  const data = await res.json();
  
  data.models.forEach(m => {
    console.log(`${m.name}: ${m.successRate}% (${m.statusCode})`);
  });
}

getStatus();
```

### Python / requests

```python
import requests

res = requests.get('http://localhost:3456/api/status')
data = res.json()

for m in data['models']:
    print(f"{m['name']}: {m['successRate']}% ({m['statusCode']})")
```

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Model struct {
    Name       string  `json:"name"`
    StatusCode string  `json:"statusCode"`
    SuccessRate float64 `json:"successRate"`
}

type Response struct {
    Models []Model `json:"models"`
}

func main() {
    resp, _ := http.Get("http://localhost:3456/api/status")
    defer resp.Body.Close()
    
    var data Response
    json.NewDecoder(resp.Body).Decode(&data)
    
    for _, m := range data.Models {
        fmt.Printf("%s: %.1f%% (%s)\n", m.Name, m.SuccessRate, m.StatusCode)
    }
}
```

---

## 部署说明

### 环境要求

- Node.js >= 18
- 系统已安装 Chrome（Windows 默认路径：`C:\Program Files\Google\Chrome\Application\chrome.exe`）

### 安装启动

```bash
# 1. 安装依赖
npm install

# 2. 启动服务
node server-api.js

# 3. 后台运行（Linux/Mac）
nohup node server-api.js > server.log 2>&1 &

# 4. 后台运行（Windows Git Bash）
nohup node server-api.js > server.log 2>&1 &
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `STATUS_PORT` | `3456` | 服务监听端口 |
| `ALLOWED_ORIGINS` | `*` | 允许的跨域来源，多个用逗号分隔，如 `https://a.com,https://b.com` |

### 服务文件说明

```
status-monitor/
├── server-api.js     # API 服务入口（只提供 JSON 接口）
├── server.js         # 带前端页面的版本（可选）
├── package.json      # 依赖配置
├── status.json       # 数据缓存（自动生成的本地缓存）
└── API.md            # 本对接文档
```

---

## 故障排查

| 现象 | 原因 | 解决 |
|------|------|------|
| 返回 `{"error":"数据尚未获取"}` | 首次启动，抓取还没完成 | 等待 5-10 秒后重试 |
| 抓取失败日志 | Playwright 找不到 Chrome | 修改 `server-api.js` 中的 `CHROME_PATH` 为正确的 Chrome 路径 |
| 端口冲突 | 3456 被占用 | `STATUS_PORT=8080 node server-api.js` |

---

## 数据来源声明

本服务抓取的数据来源于 **https://geminigen.ai/status/**，所有数据版权归 GeminiGen.AI 所有。本服务仅做数据聚合和接口封装，不做任何数据修改。
