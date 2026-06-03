# 会员订阅系统接口文档

> 基于 new-api 项目代码整理，包含用户端、管理端、支付端全部接口。

---

## 一、数据模型

### 1. SubscriptionPlan（订阅套餐）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 套餐ID |
| `title` | string | 套餐标题 |
| `subtitle` | string | 副标题 |
| `price_amount` | float64 | 价格金额 |
| `currency` | string | 币种（固定 USD） |
| `duration_unit` | string | 时长单位：year/month/day/hour/custom |
| `duration_value` | int | 时长数值 |
| `custom_seconds` | int64 | 自定义时长（秒） |
| `enabled` | bool | 是否启用 |
| `sort_order` | int | 排序权重 |
| `stripe_price_id` | string | Stripe 价格ID |
| `creem_product_id` | string | Creem 产品ID |
| `max_purchase_per_user` | int | 每人最大购买次数（0=无限） |
| `upgrade_group` | string | 购买后升级到的用户分组 |
| `total_amount` | int64 | 总额度（0=不限额度，纯会员模式） |
| `quota_reset_period` | string | 额度重置周期：never/daily/weekly/monthly/custom |
| `quota_reset_custom_seconds` | int64 | 自定义重置周期（秒） |

> **关键逻辑**：`total_amount == 0` 表示**纯会员身份**，不涉及额度限制，预扣时直接跳过额度操作。

### 2. UserSubscription（用户订阅实例）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 订阅ID |
| `user_id` | int | 用户ID |
| `plan_id` | int | 套餐ID |
| `amount_total` | int64 | 总额度（快照自套餐） |
| `amount_used` | int64 | 已用额度 |
| `start_time` | int64 | 开始时间戳 |
| `end_time` | int64 | 结束时间戳 |
| `status` | string | 状态：active/expired/cancelled |
| `source` | string | 来源：order/admin |
| `last_reset_time` | int64 | 上次额度重置时间 |
| `next_reset_time` | int64 | 下次额度重置时间 |
| `upgrade_group` | string | 升级到的分组 |
| `prev_user_group` | string | 升级前原分组（过期后回退用） |

### 3. SubscriptionOrder（订阅订单）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 订单ID |
| `user_id` | int | 用户ID |
| `plan_id` | int | 套餐ID |
| `money` | float64 | 支付金额 |
| `trade_no` | string | 商户订单号（唯一） |
| `payment_method` | string | 支付方式 |
| `status` | string | 状态：pending/success/expired |
| `create_time` | int64 | 创建时间 |
| `complete_time` | int64 | 完成时间 |
| `provider_payload` | string | 支付平台回调原始数据 |

---

## 二、用户端接口（需登录）

### 2.1 获取套餐列表

```
GET /api/subscription/plans
```

**响应**：
```json
{
  "success": true,
  "data": [
    {
      "plan": {
        "id": 1,
        "title": "月度会员",
        "subtitle": "享受全部模型优惠价格",
        "price_amount": 19.90,
        "currency": "USD",
        "duration_unit": "month",
        "duration_value": 1,
        "total_amount": 0,
        "upgrade_group": "vip",
        ...
      }
    }
  ]
}
```

---

### 2.2 获取我的订阅

```
GET /api/subscription/self
```

**响应**：
```json
{
  "success": true,
  "data": {
    "billing_preference": "subscription",
    "subscriptions": [
      {
        "subscription": {
          "id": 10,
          "user_id": 5,
          "plan_id": 1,
          "amount_total": 0,
          "amount_used": 0,
          "start_time": 1717142400,
          "end_time": 1719820800,
          "status": "active",
          "upgrade_group": "vip"
        }
      }
    ],
    "all_subscriptions": [...]
  }
}
```

---

### 2.3 更新计费偏好

```
PUT /api/subscription/self/preference
```

**请求体**：
```json
{
  "billing_preference": "subscription"
}
```

> 可选值：`subscription`（优先使用订阅） / `quota`（优先使用额度）

---

### 2.4 Epay 支付下单

```
POST /api/subscription/epay/pay
```

**请求体**：
```json
{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

**响应**：
```json
{
  "message": "success",
  "data": { /* 支付参数 */ },
  "url": "https://epay.xxx.com/submit.php?..."
}
```

---

### 2.5 Stripe 支付下单

```
POST /api/subscription/stripe/pay
```

**请求体**：
```json
{
  "plan_id": 1
}
```

**响应**：
```json
{
  "message": "success",
  "data": {
    "pay_link": "https://checkout.stripe.com/..."
  }
}
```

---

### 2.6 Creem 支付下单

```
POST /api/subscription/creem/pay
```

**请求体**：
```json
{
  "plan_id": 1
}
```

**响应**：
```json
{
  "message": "success",
  "data": {
    "checkout_url": "https://checkout.creem.io/...",
    "order_id": "sub_ref_xxx"
  }
}
```

---

## 三、管理端接口（需管理员权限）

### 3.1 套餐管理

```
GET    /api/subscription/admin/plans              # 获取全部套餐列表
POST   /api/subscription/admin/plans              # 创建套餐
PUT    /api/subscription/admin/plans/:id          # 更新套餐
PATCH  /api/subscription/admin/plans/:id          # 启停套餐（body: {"enabled": true/false}）
```

**创建/更新套餐请求体**：
```json
{
  "plan": {
    "title": "月度会员",
    "subtitle": "享受全部模型优惠",
    "price_amount": 19.90,
    "duration_unit": "month",
    "duration_value": 1,
    "total_amount": 0,
    "upgrade_group": "vip",
    "max_purchase_per_user": 0,
    "quota_reset_period": "never",
    "enabled": true,
    "sort_order": 100
  }
}
```

> `total_amount: 0` = 纯会员模式，不涉及额度限制。

---

### 3.2 为用户绑定订阅（无需支付）

```
POST /api/subscription/admin/bind
```

**请求体**：
```json
{
  "user_id": 5,
  "plan_id": 1
}
```

---

### 3.3 查询用户订阅记录

```
GET /api/subscription/admin/users/:id/subscriptions
```

---

### 3.4 为用户创建订阅（无需支付）

```
POST /api/subscription/admin/users/:id/subscriptions
```

**请求体**：
```json
{
  "plan_id": 1
}
```

---

### 3.5 作废用户订阅（立即失效）

```
POST /api/subscription/admin/user_subscriptions/:id/invalidate
```

> 状态变为 `cancelled`，`end_time` 设为当前时间，自动回退用户分组。

---

### 3.6 删除用户订阅（硬删除）

```
DELETE /api/subscription/admin/user_subscriptions/:id
```

> 数据直接删除，自动回退用户分组。

---

## 四、支付回调接口（无需登录）

### 4.1 Epay 异步通知

```
POST /api/subscription/epay/notify
GET  /api/subscription/epay/notify
```

**响应**：`success` 或 `fail`

### 4.2 Epay 同步回调

```
GET  /api/subscription/epay/return
POST /api/subscription/epay/return
```

**行为**：验证后重定向到 `/console/topup?pay=success|fail|pending`

---

## 五、核心业务流程

### 5.1 购买流程

```
用户选择套餐 → 下单创建 SubscriptionOrder(pending) → 跳转支付 → 
支付平台回调 → CompleteSubscriptionOrder → 创建 UserSubscription(active) → 
升级用户分组 → 记录日志
```

### 5.2 纯会员模式（AmountTotal == 0）

```
预扣额度时 → 检测到 AmountTotal == 0 → 直接返回成功，不创建预扣记录
```

代码位置：`model/subscription.go:1012-1018`

```go
if sub.AmountTotal == 0 {
    // 无限额度订阅：只验证有效性，不执行额度操作
    returnValue.UserSubscriptionId = sub.Id
    returnValue.PreConsumed = 0
    returnValue.AmountTotal = 0
    returnValue.AmountUsedBefore = sub.AmountUsed
    returnValue.AmountUsedAfter = sub.AmountUsed
    return nil
}
```

### 5.3 订阅过期检查

```
定时任务 ExpireDueSubscriptions → 标记过期 → 检查是否还有其他 active 订阅 → 
无则回退用户分组到 prev_user_group
```

### 5.4 额度重置

```
定时任务 ResetDueSubscriptions → 检查 next_reset_time → 
重置 amount_used = 0 → 更新 last_reset_time / next_reset_time
```

---

## 六、下游对接建议

### 判断用户是否为会员

```
GET /api/subscription/self
```

检查 `subscriptions` 数组中是否存在 `status == "active"` 的条目。

### 判断用户会员分组

订阅生效后，用户 `group` 字段会被更新为套餐配置的 `upgrade_group`。下游可通过用户分组控制模型价格和权限。

### 纯会员套餐配置示例

```json
{
  "title": "月度会员",
  "price_amount": 19.90,
  "duration_unit": "month",
  "duration_value": 1,
  "total_amount": 0,
  "upgrade_group": "vip",
  "quota_reset_period": "never"
}
```

---

## 七、相关数据库表

| 表名 | 说明 |
|------|------|
| `subscription_plans` | 套餐定义 |
| `subscription_orders` | 支付订单 |
| `user_subscriptions` | 用户订阅实例 |
| `subscription_pre_consume_records` | 额度预扣记录（幂等） |
