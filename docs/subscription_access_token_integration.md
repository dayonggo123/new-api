# 会员订阅系统 - 下游对接文档（Access Token 方式）

> 订阅接口使用 `UserAuth()` 认证，需要**用户 Access Token** + **New-Api-User Header**，不支持 API Key。

---

## 一、获取认证信息

### 1. 获取 Access Token

登录 new-api 后台 → 点击右上角头像 → **个人设置** → 复制 **Access Token**

> 注意：是 **Access Token**，不是 API Key（令牌）
> - Access Token：用户级别的登录凭证，长期有效
> - API Key（sk-xxx）：调用模型接口用的，会变化

### 2. 获取 User ID

在个人设置页面或用户管理页面查看自己的 **用户 ID**（数字）。

---

## 二、调用方式

所有订阅接口需要在 Header 中同时携带：

```http
Authorization: Bearer {AccessToken}
New-Api-User: {用户ID}
```

---

## 三、接口详情

### 3.1 获取套餐列表

```http
GET https://heharse.cloud/subscription/plans
Authorization: Bearer {你的AccessToken}
New-Api-User: {你的用户ID}
```

**正确响应示例**：
```json
{
  "success": true,
  "data": [
    {
      "plan": {
        "id": 1,
        "title": "VIP",
        "price_amount": 12.90,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "total_amount": 0,
        "upgrade_group": "vip",
        "enabled": true
      }
    },
    {
      "plan": {
        "id": 2,
        "title": "SVIP",
        "price_amount": 20.90,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "total_amount": 0,
        "upgrade_group": "svip",
        "enabled": true
      }
    }
  ]
}
```

**常见错误**：
- `Unauthorized, invalid access token` → Access Token 错误或过期
- `Unauthorized, user id not provided` → 没传 `New-Api-User` header
- `Unauthorized, user id mismatch` → `New-Api-User` 和 Access Token 对应的用户不一致

---

### 3.2 查询我的订阅

```http
GET https://heharse.cloud/subscription/self
Authorization: Bearer {你的AccessToken}
New-Api-User: {你的用户ID}
```

---

### 3.3 支付下单（Epay）

```http
POST https://heharse.cloud/subscription/epay/pay
Authorization: Bearer {你的AccessToken}
New-Api-User: {你的用户ID}
Content-Type: application/json

{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

**响应示例**：
```json
{
  "message": "success",
  "data": {
    "pid": "1001",
    "type": "alipay",
    "out_trade_no": "SUBUSR5NOabc1231717142400",
    "name": "SUB:VIP",
    "money": "12.90",
    "sign": "xxx"
  },
  "url": "https://pay.xxx.com/submit.php?..."
}
```

---

### 3.4 支付下单（Stripe）

```http
POST https://heharse.cloud/subscription/stripe/pay
Authorization: Bearer {你的AccessToken}
New-Api-User: {你的用户ID}
Content-Type: application/json

{
  "plan_id": 1
}
```

---

### 3.5 支付下单（Creem）

```http
POST https://heharse.cloud/subscription/creem/pay
Authorization: Bearer {你的AccessToken}
New-Api-User: {你的用户ID}
Content-Type: application/json

{
  "plan_id": 1
}
```

---

## 四、下游代码示例

### Python

```python
import requests
import time

BASE_URL = "https://heharse.cloud"
ACCESS_TOKEN = "你的AccessToken"
USER_ID = "你的用户ID"

HEADERS = {
    "Authorization": f"Bearer {ACCESS_TOKEN}",
    "New-Api-User": USER_ID,
    "Content-Type": "application/json"
}

def get_plans():
    r = requests.get(f"{BASE_URL}/subscription/plans", headers=HEADERS)
    return r.json()

def get_my_subscription():
    r = requests.get(f"{BASE_URL}/subscription/self", headers=HEADERS)
    return r.json()

def pay_epay(plan_id, payment_method="alipay"):
    r = requests.post(
        f"{BASE_URL}/subscription/epay/pay",
        headers=HEADERS,
        json={"plan_id": plan_id, "payment_method": payment_method}
    )
    return r.json()

def is_vip():
    result = get_my_subscription()
    if not result.get("success"):
        return False, ""
    subs = result.get("data", {}).get("subscriptions", [])
    now = int(time.time())
    for item in subs:
        sub = item.get("subscription", {})
        if sub.get("status") == "active" and sub.get("end_time", 0) > now:
            return True, sub.get("upgrade_group", "")
    return False, ""
```

### JavaScript

```javascript
const BASE_URL = "https://heharse.cloud";
const ACCESS_TOKEN = "你的AccessToken";
const USER_ID = "你的用户ID";

const headers = {
  "Authorization": `Bearer ${ACCESS_TOKEN}`,
  "New-Api-User": USER_ID,
  "Content-Type": "application/json"
};

async function getPlans() {
  const res = await fetch(`${BASE_URL}/subscription/plans`, { headers });
  return res.json();
}

async function getMySubscription() {
  const res = await fetch(`${BASE_URL}/subscription/self`, { headers });
  return res.json();
}
```

### cURL 测试

```bash
# 获取套餐列表
curl -H "Authorization: Bearer 你的AccessToken" \
     -H "New-Api-User: 你的用户ID" \
     https://heharse.cloud/subscription/plans

# 查询我的订阅
curl -H "Authorization: Bearer 你的AccessToken" \
     -H "New-Api-User: 你的用户ID" \
     https://heharse.cloud/subscription/self

# Epay 下单
curl -X POST \
     -H "Authorization: Bearer 你的AccessToken" \
     -H "New-Api-User: 你的用户ID" \
     -H "Content-Type: application/json" \
     -d '{"plan_id":1,"payment_method":"alipay"}' \
     https://heharse.cloud/subscription/epay/pay
```

---

## 五、常见问题

### Q1: 为什么不能用 API Key（sk-xxx）？

订阅接口使用 `UserAuth()` 中间件，验证的是 `users.access_token`（用户 Access Token），不是 `tokens.key`（API Key）。API Key 主要用于调用模型接口。

### Q2: Access Token 和 API Key 的区别？

| | Access Token | API Key |
|--|-------------|---------|
| 用途 | 用户登录、管理接口 | 调用模型接口 |
| 存储位置 | `users.access_token` | `tokens` 表 |
| 变化频率 | 长期有效，手动重置才变 | 可频繁创建/删除 |

### Q3: `New-Api-User` 是什么？

安全校验字段。`UserAuth()` 要求请求中必须携带 `New-Api-User` header，其值要和 Access Token 对应的用户 ID 一致，防止 Token 被盗用后冒充其他用户。
