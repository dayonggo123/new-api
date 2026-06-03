# New-API 订阅系统 API 对接文档（下游对接版）

> 版本：v1.1
> 日期：2026-05-30
> Base URL：`https://<your-domain>/api`
> 认证方式：Header `Authorization: Bearer <Session-Key>`

---

## 目录

1. [认证说明](#认证说明)
2. [通用响应格式](#通用响应格式)
3. [用户端接口](#用户端接口)
   - 获取套餐列表
   - 获取自己的订阅
   - 更新计费偏好
   - 发起支付（Epay）
   - 发起支付（Stripe）
   - 发起支付（Creem）
4. [支付回调说明](#支付回调说明)
5. [数据结构](#数据结构)
6. [错误码](#错误码)
7. [对接流程示例](#对接流程示例)

---

## 认证说明

所有用户端接口需在 HTTP Header 中携带 Session Key：

```http
Authorization: Bearer sk-xxxxxxxxxxxxxxxxxxxxxxxx
```

Session Key 即 new-api 用户的 API Key。

---

## 通用响应格式

```json
{
  "success": true,
  "message": "",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `success` | boolean | `true` 成功，`false` 失败 |
| `message` | string | 错误时返回提示信息 |
| `data` | any | 成功时返回具体数据 |

---

## 用户端接口

### 1. 获取订阅套餐列表

获取所有启用的订阅套餐。

```http
GET /api/subscription/plans
Authorization: Bearer <Session-Key>
```

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": [
    {
      "plan": {
        "id": 1,
        "title": "Pro会员",
        "subtitle": "高级模型无限畅用",
        "price_amount": 29.99,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "custom_seconds": 0,
        "enabled": true,
        "sort_order": 10,
        "stripe_price_id": "price_xxx",
        "creem_product_id": "",
        "max_purchase_per_user": 0,
        "total_amount": 0,
        "upgrade_group": "vip",
        "quota_reset_period": "never",
        "quota_reset_custom_seconds": 0,
        "created_at": 1717000000,
        "updated_at": 1717000000
      }
    }
  ]
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | int | 套餐 ID |
| `title` | string | 套餐标题 |
| `subtitle` | string | 副标题/描述 |
| `price_amount` | float64 | 价格（单位：`currency`） |
| `currency` | string | 货币，固定 `USD` |
| `duration_unit` | string | 周期单位：`year`/`month`/`day`/`hour`/`custom` |
| `duration_value` | int | 周期数值 |
| `total_amount` | int64 | **订阅额度总量（0 = 无限/纯会员）** |
| `upgrade_group` | string | 购买后升级到的用户组 |
| `quota_reset_period` | string | 额度重置周期：`never`/`daily`/`weekly`/`monthly`/`custom` |
| `max_purchase_per_user` | int | 每用户最大购买次数（0 = 无限） |

> **重要**：当 `total_amount == 0` 时，该套餐为**纯会员套餐**，只用于升级用户分组（`upgrade_group`），不涉及额度扣除。用户购买后只享受分组权益（模型访问、价格等），调用 API 时不扣订阅额度。

---

### 2. 获取自己的订阅

获取当前用户的所有订阅（含已过期）。

```http
GET /api/subscription/self
Authorization: Bearer <Session-Key>
```

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "billing_preference": "subscription",
    "subscriptions": [
      {
        "subscription": {
          "id": 1,
          "user_id": 42,
          "plan_id": 1,
          "amount_total": 0,
          "amount_used": 0,
          "start_time": 1717000000,
          "end_time": 1719592000,
          "status": "active",
          "source": "order",
          "last_reset_time": 0,
          "next_reset_time": 0,
          "upgrade_group": "vip",
          "prev_user_group": "default"
        }
      }
    ],
    "all_subscriptions": [
      // 包含已过期/已取消的订阅
    ]
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|---|---|---|
| `billing_preference` | string | 计费偏好：`subscription`/`quota`/`auto` |
| `subscriptions` | array | 当前有效（active）的订阅 |
| `all_subscriptions` | array | 全部订阅（含过期） |
| `amount_total` | int64 | 总额度（0 = 无限） |
| `amount_used` | int64 | 已用额度 |
| `status` | string | `active`/`expired`/`cancelled` |
| `upgrade_group` | string | 该订阅绑定的用户组 |

---

### 3. 更新计费偏好

设置用户优先使用订阅额度还是余额额度。

```http
PUT /api/subscription/self/preference
Authorization: Bearer <Session-Key>
Content-Type: application/json
```

**请求体：**

```json
{
  "billing_preference": "subscription"
}
```

| 参数 | 类型 | 说明 |
|---|---|---|
| `billing_preference` | string | `subscription` 优先订阅 / `quota` 优先余额 / `auto` 自动 |

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "billing_preference": "subscription"
  }
}
```

---

### 4. 发起支付 - Epay（易支付）

```http
POST /api/subscription/epay/pay
Authorization: Bearer <Session-Key>
Content-Type: application/json
```

**请求体：**

```json
{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `plan_id` | int | 是 | 套餐 ID |
| `payment_method` | string | 是 | 支付方式：`alipay`/`wxpay`/`qqpay` 等 |

**响应：**

```json
{
  "message": "success",
  "data": {
    "pid": "1000",
    "type": "alipay",
    "out_trade_no": "SUBUSR42NOabc1231717000000",
    "notify_url": "https://your-domain/api/subscription/epay/notify",
    "return_url": "https://your-domain/api/subscription/epay/return",
    "name": "SUB:Pro会员",
    "money": "29.99",
    "sign": "xxx",
    "sign_type": "MD5"
  },
  "url": "https://pay.example.com/submit.php"
}
```

| 字段 | 说明 |
|---|---|
| `data` | 支付表单参数，需以 `POST` 或 `GET` 方式提交到 `url` |
| `url` | 支付网关地址 |

> **对接方式**：拿到 `data` 和 `url` 后，可以构造表单跳转或直接用 `POST` 方式提交到 `url`。

---

### 5. 发起支付 - Stripe

```http
POST /api/subscription/stripe/pay
Authorization: Bearer <Session-Key>
Content-Type: application/json
```

**请求体：**

```json
{
  "plan_id": 1
}
```

**响应：**

```json
{
  "message": "success",
  "data": {
    "pay_link": "https://checkout.stripe.com/c/pay/cs_test_xxx"
  }
}
```

> 拿到 `pay_link` 后，引导用户跳转到 Stripe Checkout 完成支付。

---

### 6. 发起支付 - Creem

```http
POST /api/subscription/creem/pay
Authorization: Bearer <Session-Key>
Content-Type: application/json
```

**请求体：**

```json
{
  "plan_id": 1
}
```

**响应：**

```json
{
  "message": "success",
  "data": {
    "checkout_url": "https://checkout.creem.io/pay_xxx",
    "order_id": "sub_ref_xxx"
  }
}
```

> 拿到 `checkout_url` 后，引导用户跳转到 Creem Checkout 完成支付。

---

## 支付回调说明

### Epay 异步通知

Epay 支付完成后会异步 POST 到该地址，**无需下游处理**，由 new-api 服务端自动完成。

```http
POST /api/subscription/epay/notify
GET  /api/subscription/epay/notify
```

### Epay 同步返回

用户支付完成后浏览器会重定向到该地址。

```http
GET /api/subscription/epay/return
POST /api/subscription/epay/return
```

### Stripe Webhook

Stripe 支付完成后会通过 Webhook 通知，**无需下游处理**。

### Creem Webhook

Creem 支付完成后会通过 Webhook 通知，**无需下游处理**。

> **下游确认支付状态的方式**：在前端轮询 `GET /api/subscription/self`，检查 `subscriptions` 中是否新增 `active` 记录。

---

## 数据结构

### SubscriptionPlan（订阅套餐）

```typescript
interface SubscriptionPlan {
  id: number;
  title: string;
  subtitle: string;
  price_amount: number;
  currency: string;
  duration_unit: "year" | "month" | "day" | "hour" | "custom";
  duration_value: number;
  custom_seconds: number;
  enabled: boolean;
  sort_order: number;
  stripe_price_id: string;
  creem_product_id: string;
  max_purchase_per_user: number;
  total_amount: number;      // 额度总量，0 = 无限/纯会员
  upgrade_group: string;     // 购买后升级到的用户组
  quota_reset_period: "never" | "daily" | "weekly" | "monthly" | "custom";
  quota_reset_custom_seconds: number;
  created_at: number;
  updated_at: number;
}
```

### UserSubscription（用户订阅实例）

```typescript
interface UserSubscription {
  id: number;
  user_id: number;
  plan_id: number;
  amount_total: number;      // 0 = 无限额度
  amount_used: number;
  start_time: number;        // 秒级时间戳
  end_time: number;
  status: "active" | "expired" | "cancelled";
  source: "order" | "admin";
  last_reset_time: number;
  next_reset_time: number;
  upgrade_group: string;
  prev_user_group: string;
}
```

---

## 错误码

| HTTP 状态 | 场景 | 说明 |
|---|---|---|
| 401 | 未登录/Session Key 无效 | 检查 Authorization Header |
| 400 | 参数错误 | `plan_id` 无效或缺失 |
| 403 | 套餐未启用 | 该套餐已被管理员下架 |
| 403 | 已达到购买上限 | `max_purchase_per_user` 限制 |
| 400 | 套餐金额过低 | `price_amount < 0.01` |
| 400 | 支付方式不存在 | `payment_method` 不在配置中 |
| 400 | 该套餐未配置 StripePriceId | Stripe 支付专用 |
| 400 | 该套餐未配置 CreemProductId | Creem 支付专用 |
| 500 | 服务端错误 | 联系管理员 |

---

## 对接流程示例

### 场景：用户购买月度 Pro 会员（纯会员模式）

```
1. 用户打开会员页面
   └─> GET /api/subscription/plans
       └─> 展示套餐列表（标题、价格、权益）
       └─> 发现 "Pro会员" total_amount=0，说明是纯会员

2. 用户选择「Pro会员」点击购买
   └─> POST /api/subscription/epay/pay
       Body: { "plan_id": 1, "payment_method": "alipay" }
       └─> 拿到 data + url，构造支付请求

3. 用户完成支付，返回应用
   └─> 前端轮询 GET /api/subscription/self
       └─> 发现 subscriptions 中新增 active 记录
       └─> 用户 group 已变为 "vip"

4. 用户调用 API
   └─> 使用原有 OpenAI 兼容接口
       └─> total_amount=0 的订阅不扣额度
       └─> 享受 vip 分组的模型和价格
```

### 分组权益说明

| 用户组 | 权益来源 |
|---|---|
| `default` | 默认分组，所有注册用户 |
| `vip`/`svip` 等 | 通过订阅 `upgrade_group` 自动升级 |

分组决定：
- **可用模型**：由 `abilities` 表中该分组关联的渠道决定
- **调用价格**：由 `group_ratio` 和 `group_group_ratio` 配置决定
- **额度扣除**：纯会员套餐（`total_amount == 0`）不扣除任何额度

---

## 后端管理接口（Admin）

如需对接管理后台，使用以下接口（需 Admin 权限）：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/subscription/admin/plans` | 列出所有套餐（含禁用） |
| POST | `/api/subscription/admin/plans` | 创建套餐 |
| PUT | `/api/subscription/admin/plans/:id` | 更新套餐 |
| PATCH | `/api/subscription/admin/plans/:id` | 启用/禁用套餐 |
| POST | `/api/subscription/admin/bind` | 为用户绑定订阅（免支付） |
| GET | `/api/subscription/admin/users/:id/subscriptions` | 查看用户订阅 |
| POST | `/api/subscription/admin/users/:id/subscriptions` | 为用户创建订阅 |
| POST | `/api/subscription/admin/user_subscriptions/:id/invalidate` | 取消订阅 |
| DELETE | `/api/subscription/admin/user_subscriptions/:id` | 删除订阅 |
