# 用户注册接口文档

## 接口概述

| 属性 | 说明 |
|------|------|
| **接口地址** | `POST /api/user/register` |
| **Content-Type** | `application/json` |
| **认证方式** | 无需认证（公开接口） |
| **限流策略** | CriticalRateLimit（严格限流） |
| **人机验证** | Turnstile（如系统开启） |

---

## 请求参数

### Body 参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 用户名，需通过系统校验规则 |
| `password` | string | 是 | 密码，建议 8-20 位 |
| `email` | string | 条件必填 | 如系统开启邮箱验证则必填 |
| `verification_code` | string | 条件必填 | 邮箱验证码，与 `email` 配对使用 |
| `aff_code` | string | 否 | 邀请人邀请码，用于绑定上下级关系 |

### Query 参数

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `turnstile` | string | 条件必填 | Turnstile 验证令牌，如系统开启 Turnstile 则必填 |

---

## 请求示例

### cURL

```bash
curl -X POST "https://your-domain.com/api/user/register?turnstile=YOUR_TURNSTILE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "YourPassword123",
    "email": "user@example.com",
    "verification_code": "123456",
    "aff_code": "ABCD"
  }'
```

### JavaScript (Fetch)

```javascript
const response = await fetch('/api/user/register?turnstile=turnstileToken', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username: 'testuser',
    password: 'YourPassword123',
    email: 'user@example.com',
    verification_code: '123456',
    aff_code: 'ABCD'
  })
});
const data = await response.json();
```

### Python (Requests)

```python
import requests

response = requests.post(
    'https://your-domain.com/api/user/register',
    params={'turnstile': 'turnstileToken'},
    json={
        'username': 'testuser',
        'password': 'YourPassword123',
        'email': 'user@example.com',
        'verification_code': '123456',
        'aff_code': 'ABCD'
    }
)
data = response.json()
```

### Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RegisterRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	Email            string `json:"email,omitempty"`
	VerificationCode string `json:"verification_code,omitempty"`
	AffCode          string `json:"aff_code,omitempty"`
}

func main() {
	reqBody := RegisterRequest{
		Username:         "testuser",
		Password:         "YourPassword123",
		Email:            "user@example.com",
		VerificationCode: "123456",
		AffCode:          "ABCD",
	}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		"https://your-domain.com/api/user/register?turnstile=token",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println(result)
}
```

---

## 响应格式

### 成功响应 (HTTP 200)

```json
{
  "success": true,
  "message": ""
}
```

### 失败响应 (HTTP 200，业务错误)

```json
{
  "success": false,
  "message": "错误描述信息"
}
```

---

## 错误码说明

| 场景 | HTTP 状态 | 响应 message |
|------|-----------|--------------|
| 注册功能已关闭 | 200 | `用户注册已关闭` |
| 密码注册已关闭 | 200 | `密码注册已关闭` |
| 请求参数错误 | 200 | `参数错误` |
| 输入验证失败 | 200 | `用户输入不合法: {具体错误}` |
| 邮箱验证未通过 | 200 | `邮箱验证失败` |
| 验证码错误 | 200 | `验证码错误` |
| 用户名/邮箱已存在 | 200 | `用户已存在` |
| 数据库错误 | 200 | `数据库错误` |
| 注册失败 | 200 | `注册失败` |
| 默认令牌创建失败 | 200 | `创建默认令牌失败` |
| Turnstile 验证失败 | 403 / 429 | 由中间件返回 |
| 请求过于频繁 | 429 | 由限流中间件返回 |

---

## 后端处理逻辑

1. **开关检查**：确认 `RegisterEnabled` 和 `PasswordRegisterEnabled` 均为 true
2. **参数解析 & 校验**：解析请求体，对 `User` 结构体进行 validate
3. **邮箱验证**（如开启）：校验 `email` 和 `verification_code` 是否匹配
4. **用户存在性检查**：检查 `username` 或 `email` 是否已存在（含软删除）
5. **创建用户**：生成干净的用户对象，设置默认角色为普通用户 (`RoleCommonUser`)，执行 `Insert()`
6. **创建默认令牌**（如开启 `GenerateDefaultToken`）：
   - 令牌名称：`{用户名}的初始令牌`
   - 额度：`500000`
   - 永不过期 (`ExpiredTime: -1`)
7. **返回成功响应**

---

## 相关配置项

| 配置项 | 说明 | 影响 |
|--------|------|------|
| `RegisterEnabled` | 是否允许注册 | 关闭时接口直接拒绝 |
| `PasswordRegisterEnabled` | 是否允许密码注册 | 关闭时接口直接拒绝 |
| `EmailVerificationEnabled` | 是否强制邮箱验证 | 开启时 `email` 和 `verification_code` 必填 |
| `GenerateDefaultToken` | 是否自动生成默认令牌 | 开启时注册成功后自动创建初始 API Key |
| `TurnstileCheck` | 是否开启人机验证 | 开启时 URL 需带 `?turnstile=` |

---

## 前端注册页面

- **路径**：`/register`
- **组件**：`web/src/components/auth/RegisterForm.jsx`
- **特性**：
  - 支持用户名密码注册
  - 支持邮箱验证码（如开启）
  - 支持 OAuth 注册（GitHub / Discord / OIDC / LinuxDO / 微信 / Telegram / 自定义 OAuth）
  - 支持 Turnstile 人机验证
  - 支持用户协议和隐私政策勾选

---

## 路由定义

文件：`router/api-router.go`

```go
userRoute.POST("/register",
    middleware.CriticalRateLimit(),
    middleware.TurnstileCheck(),
    controller.Register,
)
```

---

*文档生成时间：2026-06-01*
