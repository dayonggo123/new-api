# HeHarseCloud 会员订阅接口文档

> 接口域名：`https://heharse.cloud`
> 认证方式：Access Token + New-Api-User（**不是 API Key**）

---

## 一、认证说明

### 注意：不能用 API Key（sk-xxx）

订阅接口使用 `UserAuth()` 认证，必须同时提供：

| Header | 说明 | 获取方式 |
|--------|------|---------|
| `Authorization` | `Bearer {AccessToken}` | 后台 **个人设置** 页面，复制 Access Token |
| `New-Api-User` | 用户 ID（数字） | 后台 **个人设置** 页面查看 User ID |

### 认证失败常见原因

| 现象 | 原因 |
|------|------|
| `401 Unauthorized` | 没传 `New-Api-User` header |
| `用户ID不匹配` | Access Token 和 `New-Api-User` 不是同一个用户 |
| `Access Token 无效` | 用了 API Key（sk-xxx）而不是 Access Token |

---

## 二、接口列表

### 1. 获取订阅套餐列表

获取当前可用的所有会员套餐。

```http
GET /api/subscription/plans
```

**请求头：**
```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
```

**响应示例：**
```json
{
  "success": true,
  "data": [
    {
      "plan": {
        "id": 2,
        "title": "SVIP",
        "subtitle": "",
        "description": "",
        "price_amount": 20.90,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "total_amount": 0,
        "reset_strategy": "monthly",
        "upgrade_group": "svip",
        "enabled": true,
        "sort_order": 0
      }
    },
    {
      "plan": {
        "id": 1,
        "title": "VIP",
        "subtitle": "",
        "description": "",
        "price_amount": 12.90,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "total_amount": 0,
        "reset_strategy": "monthly",
        "upgrade_group": "vip",
        "enabled": true,
        "sort_order": 0
      }
    }
  ]
}
```

**关键字段说明：**

| 字段 | 说明 |
|------|------|
| `id` | 套餐 ID，下单时需要传 |
| `title` | 套餐名称（VIP / SVIP） |
| `price_amount` | 价格（美元） |
| `duration_unit` + `duration_value` | 有效期（如 `month` + `1` = 1个月） |
| `total_amount` | **0 表示不限额度**（纯会员模式） |
| `reset_strategy` | 额度重置策略：`monthly` 每月重置 |
| `upgrade_group` | 购买后升级到的用户分组（`vip` / `svip`） |

---

### 2. 创建支付订单（Epay - 支付宝/微信）

```http
POST /api/subscription/epay/pay
```

**请求头：**
```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
Content-Type: application/json
```

**请求体：**
```json
{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `plan_id` | int | 是 | 套餐 ID（从 `/plans` 接口获取） |
| `payment_method` | string | 是 | 支付方式：`alipay`（支付宝）或 `wxpay`（微信） |

**响应示例（成功）：**
```json
{
  "success": true,
  "message": "ok",
  "data": "https://pay.example.com/pay?trade_no=SUB-20260531-xxxx"
}
```

**响应中的 `data` 是支付跳转链接**，引导用户前往支付宝/微信完成支付。

**响应示例（失败）：**
```json
{
  "success": false,
  "message": "套餐未启用"
}
```

---

### 3. 创建支付订单（Stripe）

```http
POST /api/subscription/stripe/pay
```

**请求头：**
```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
Content-Type: application/json
```

**请求体：**
```json
{
  "plan_id": 1
}
```

**响应示例（成功）：**
```json
{
  "success": true,
  "message": "ok",
  "data": "https://checkout.stripe.com/pay/cs_test_xxx"
}
```

**响应中的 `data` 是 Stripe Checkout 链接。**

> **注意**：Stripe 需要在后台配置套餐的 `stripe_price_id`（Stripe Price ID，格式 `price_xxx`），否则返回错误。

---

### 4. 创建支付订单（Creem）

```http
POST /api/subscription/creem/pay
```

**请求头：**
```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
Content-Type: application/json
```

**请求体：**
```json
{
  "plan_id": 1
}
```

**响应示例（成功）：**
```json
{
  "success": true,
  "message": "ok",
  "data": "https://checkout.creem.io/checkout_xxx"
}
```

> **注意**：Creem 需要在后台配置套餐的 `creem_product_id`（Creem Product ID，格式 `prod_xxx`），否则返回错误。

---

### 5. 查询我的订阅

查询当前用户的所有订阅记录（包括已过期的）。

```http
GET /api/subscription/self
```

**请求头：**
```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
```

**响应示例：**
```json
{
  "success": true,
  "message": "ok",
  "data": {
    "active_subscription": {
      "id": 1,
      "plan_id": 1,
      "plan_name": "VIP",
      "amount_total": 0,
      "amount_used": 0,
      "start_time": "2026-05-31T00:00:00Z",
      "end_time": "2026-06-30T23:59:59Z",
      "status": "active"
    },
    "all_subscriptions": [
      {
        "id": 1,
        "plan_id": 1,
        "plan_name": "VIP",
        "amount_total": 0,
        "amount_used": 0,
        "start_time": "2026-05-31T00:00:00Z",
        "end_time": "2026-06-30T23:59:59Z",
        "status": "active"
      }
    ],
    "billing_preference": "subscription_first"
  }
}
```

**关键字段说明：**

| 字段 | 说明 |
|------|------|
| `active_subscription` | 当前生效的订阅（如果没有则为空） |
| `status` | `active`（生效中）、`expired`（已过期）、`cancelled`（已取消） |
| `amount_total` | **0 表示不限额度** |
| `start_time` / `end_time` | 订阅有效期 |

---

## 三、完整对接流程

### 时序图

```
┌─────────┐     ┌─────────────┐     ┌──────────────┐
│  用户   │     │  下游应用   │     │ HeHarseCloud │
└────┬────┘     └──────┬──────┘     └──────┬───────┘
     │                 │                    │
     │  打开会员页面   │                    │
     │────────────────>│                    │
     │                 │ GET /subscription/plans
     │                 │───────────────────>│
     │                 │ 返回 VIP/SVIP 套餐 │
     │                 │<───────────────────│
     │  展示套餐价格   │                    │
     │<────────────────│                    │
     │                 │                    │
     │  选择 VIP 并支付 │                    │
     │────────────────>│                    │
     │                 │ POST /subscription/epay/pay
     │                 │ {plan_id: 1}       │
     │                 │───────────────────>│
     │                 │ 返回支付链接        │
     │                 │<───────────────────│
     │  跳转支付页面   │                    │
     │<────────────────│                    │
     │                 │                    │
     │  完成支付       │                    │
     │────────────────>│                    │
     │                 │ 支付平台回调       │
     │                 │ 后台激活订阅       │
     │                 │                    │
     │  刷新会员状态   │                    │
     │────────────────>│                    │
     │                 │ GET /subscription/self
     │                 │───────────────────>│
     │                 │ 返回 active 订阅   │
     │                 │<───────────────────│
     │  显示 VIP 会员  │                    │
     │<────────────────│                    │
```

---

## 四、Python 对接示例

```python
import requests

BASE_URL = "https://heharse.cloud/api"
ACCESS_TOKEN = "你的AccessToken"  # 从后台个人设置复制
USER_ID = "你的用户ID"            # 从后台个人设置查看

def get_headers():
    return {
        "Authorization": f"Bearer {ACCESS_TOKEN}",
        "New-Api-User": USER_ID,
        "Content-Type": "application/json"
    }

# 1. 获取套餐列表
response = requests.get(f"{BASE_URL}/subscription/plans", headers=get_headers())
plans = response.json()
print("套餐列表:", plans)

# 2. 创建支付订单（以 Epay 支付宝为例）
response = requests.post(
    f"{BASE_URL}/subscription/epay/pay",
    headers=get_headers(),
    json={"plan_id": 1, "payment_method": "alipay"}
)
result = response.json()
if result.get("success"):
    pay_url = result["data"]
    print(f"支付链接: {pay_url}")
    # 引导用户浏览器跳转到 pay_url
else:
    print(f"创建订单失败: {result.get('message')}")

# 3. 查询我的订阅
response = requests.get(f"{BASE_URL}/subscription/self", headers=get_headers())
my_sub = response.json()
print("我的订阅:", my_sub)
```

---

## 五、JavaScript 对接示例

```javascript
const BASE_URL = "https://heharse.cloud/api";
const ACCESS_TOKEN = "你的AccessToken";
const USER_ID = "你的用户ID";

function getHeaders() {
  return {
    "Authorization": `Bearer ${ACCESS_TOKEN}`,
    "New-Api-User": USER_ID,
    "Content-Type": "application/json"
  };
}

// 1. 获取套餐列表
fetch(`${BASE_URL}/subscription/plans`, {
  method: "GET",
  headers: getHeaders()
})
.then(res => res.json())
.then(data => console.log("套餐列表:", data));

// 2. 创建支付订单
fetch(`${BASE_URL}/subscription/epay/pay`, {
  method: "POST",
  headers: getHeaders(),
  body: JSON.stringify({ plan_id: 1, payment_method: "alipay" })
})
.then(res => res.json())
.then(data => {
  if (data.success) {
    window.location.href = data.data; // 跳转支付
  } else {
    console.error("创建订单失败:", data.message);
  }
});
```

---

## 六、常见问题

### Q: 为什么返回 "Unauthorized, invalid access token"？

A: 你用了 API Key（sk-xxx）。订阅接口必须用 **Access Token**（从后台个人设置页面复制）。

### Q: 为什么返回 "用户ID未提供"？

A: 必须加 `New-Api-User` header，且值是数字格式的用户 ID。

### Q: 为什么返回 "用户ID不匹配"？

A: `New-Api-User` 的值和 `Authorization` 里的 Access Token 不是同一个用户。

### Q: 支付成功了为什么用户没变成会员？

A: 检查支付回调地址是否能被外网访问。Epay 需要公网能访问的回调地址才能异步通知成功。

### Q: `total_amount` 为什么是 0？

A: **0 表示不限额度**。这是纯会员模式，用户购买后只获得会员身份（享受不同价格和模型权限），不消耗额度。

### Q: 用户分组怎么用？

A: 购买 VIP 后用户 `group` 字段变成 `vip`，购买 SVIP 后变成 `svip`。下游可以用用户 `group` 来控制模型价格和功能权限。

---

## 七、后台配置检查清单

| 检查项 | 位置 |
|--------|------|
| 套餐已创建并启用 | 后台 → 订阅管理 → 编辑套餐 → 启用开关 |
| Epay 支付已配置 | 后台 → 系统设置 → 支付设置 → Epay |
| Stripe 已配置（如需要） | 后台 → 订阅管理 → 编辑套餐 → Stripe Price ID |
| Creem 已配置（如需要） | 后台 → 订阅管理 → 编辑套餐 → Creem Product ID |
| 回调地址可访问 | 确保 `https://heharse.cloud/api/subscription/epay/notify` 能被外网访问 |
