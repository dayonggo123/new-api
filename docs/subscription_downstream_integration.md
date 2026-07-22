# 会员订阅系统 — 下游对接示例

> 下游系统调用 new-api 订阅支付接口的完整对接指南。

---

## 一、接口基地址

```
https://heharse.cloud/api
```

所有请求需在 Header 中携带：
```
Authorization: Bearer {用户API Key}
```

---

## 二、完整支付流程

### Step 1: 获取套餐列表

```http
GET /subscription/plans
```

**响应示例**：
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
        "stripe_price_id": "price_xxx",
        "creem_product_id": "prod_xxx"
      }
    },
    {
      "plan": {
        "id": 2,
        "title": "年度会员",
        "subtitle": "全年尊享",
        "price_amount": 199.00,
        "currency": "USD",
        "duration_unit": "year",
        "duration_value": 1,
        "total_amount": 0,
        "upgrade_group": "svip",
        "stripe_price_id": "price_yyy",
        "creem_product_id": "prod_yyy"
      }
    }
  ]
}
```

> **关键字段说明**：
> - `id` — 套餐ID，下单时传入
> - `price_amount` — **Epay 支付时使用此金额**
> - `stripe_price_id` — **Stripe 支付时使用此ID**
> - `creem_product_id` — **Creem 支付时使用此ID**
> - `total_amount: 0` — 纯会员模式，不涉及额度

---

### Step 2: 选择支付方式下单

#### 方式 A: Epay 支付（支付宝/微信）

```http
POST /subscription/epay/pay
Content-Type: application/json

{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

> `payment_method` 可选值取决于后台配置，常见：`alipay`, `wxpay`

**响应示例**：
```json
{
  "message": "success",
  "data": {
    "pid": "1001",
    "type": "alipay",
    "out_trade_no": "SUBUSR5NOabc1231717142400",
    "notify_url": "https://heharse.cloud/api/subscription/epay/notify",
    "return_url": "https://heharse.cloud/api/subscription/epay/return",
    "name": "SUB:月度会员",
    "money": "19.90",
    "sign": "xxx",
    "sign_type": "MD5"
  },
  "url": "https://pay.xxx.com/submit.php?pid=1001&..."
}
```

**下游处理**：
1. 将用户浏览器重定向到 `url`，或
2. 用 `data` 参数构造支付表单 POST 到 Epay 网关

---

#### 方式 B: Stripe 支付

```http
POST /subscription/stripe/pay
Content-Type: application/json

{
  "plan_id": 1
}
```

**响应示例**：
```json
{
  "message": "success",
  "data": {
    "pay_link": "https://checkout.stripe.com/c/pay/..."
  }
}
```

**下游处理**：将用户浏览器重定向到 `pay_link` 完成支付。

> ⚠️ **注意**：Stripe 的金额是在 Stripe 后台配置的，代码中只传 `plan_id`，**不传金额**。

---

#### 方式 C: Creem 支付

```http
POST /subscription/creem/pay
Content-Type: application/json

{
  "plan_id": 1
}
```

**响应示例**：
```json
{
  "message": "success",
  "data": {
    "checkout_url": "https://checkout.creem.io/...",
    "order_id": "sub_ref_abc123"
  }
}
```

**下游处理**：将用户浏览器重定向到 `checkout_url` 完成支付。

> ⚠️ **注意**：Creem 的金额也是在 Creem 后台配置的，代码中只传 `plan_id`。

---

### Step 3: 支付成功后查询订阅状态

支付完成后，调用以下接口确认用户是否已成为会员：

```http
GET /subscription/self
```

**响应示例（已购买成功）**：
```json
{
  "success": true,
  "data": {
    "billing_preference": "subscription",
    "subscriptions": [
      {
        "subscription": {
          "id": 10,
          "plan_id": 1,
          "amount_total": 0,
          "amount_used": 0,
          "start_time": 1717142400,
          "end_time": 1719820800,
          "status": "active",
          "upgrade_group": "vip"
        }
      }
    ]
  }
}
```

**判断逻辑**：
```python
# 示例：判断用户是否为会员
import time

def is_vip(subscription_self_response):
    subs = subscription_self_response.get("data", {}).get("subscriptions", [])
    now = int(time.time())
    for item in subs:
        sub = item.get("subscription", {})
        if sub.get("status") == "active" and sub.get("end_time", 0) > now:
            return True, sub.get("upgrade_group", "")
    return False, ""

# 调用
is_member, group = is_vip(response)
if is_member:
    print(f"用户是会员，分组：{group}")
else:
    print("用户不是会员")
```

---

## 三、三种支付方式对比

| 支付方式 | 下单接口 | 传什么 | 金额在哪配置 |
|---------|---------|--------|-------------|
| **Epay** | `POST /subscription/epay/pay` | `plan_id` + `payment_method` | new-api 后台套餐的 `price_amount` |
| **Stripe** | `POST /subscription/stripe/pay` | `plan_id` | Stripe 后台的 Price ID |
| **Creem** | `POST /subscription/creem/pay` | `plan_id` | Creem 后台的 Product ID |

---

## 四、常见联调问题

### Q1: 下单报 "套餐金额过低"

检查套餐 `price_amount` 是否 >= 0.01。如果是 0 元套餐，Epay 不支持，需要用管理员接口直接绑定。

### Q2: Stripe 下单报 "该套餐未配置 StripePriceId"

后台编辑套餐，填入 Stripe Price ID（格式如 `price_xxx`）。

### Q3: Creem 下单报 "该套餐未配置 CreemProductId"

后台编辑套餐，填入 Creem Product ID。

### Q4: 支付成功了但用户还是非会员

检查支付回调是否正常工作：
- Epay: 查看 `/api/subscription/epay/notify` 是否收到通知
- Stripe: 检查 Webhook 配置和事件投递
- Creem: 检查 Webhook 配置

也可以调用管理员接口直接为用户绑定订阅（免支付调试）：
```http
POST /subscription/admin/bind
Authorization: Bearer {管理员API Key}

{
  "user_id": 5,
  "plan_id": 1
}
```

---

## 五、后端接入示例（Python）

```python
import requests

BASE_URL = "https://heharse.cloud/api"
HEADERS = {"Authorization": "Bearer sk-xxxx"}

def get_plans():
    """获取套餐列表"""
    r = requests.get(f"{BASE_URL}/subscription/plans", headers=HEADERS)
    return r.json()

def pay_epay(plan_id: int, payment_method: str = "alipay"):
    """Epay 支付下单"""
    r = requests.post(
        f"{BASE_URL}/subscription/epay/pay",
        headers={**HEADERS, "Content-Type": "application/json"},
        json={"plan_id": plan_id, "payment_method": payment_method}
    )
    data = r.json()
    return data.get("url"), data.get("data")

def pay_stripe(plan_id: int):
    """Stripe 支付下单"""
    r = requests.post(
        f"{BASE_URL}/subscription/stripe/pay",
        headers={**HEADERS, "Content-Type": "application/json"},
        json={"plan_id": plan_id}
    )
    return r.json().get("data", {}).get("pay_link")

def get_my_subscription():
    """查询我的订阅"""
    r = requests.get(f"{BASE_URL}/subscription/self", headers=HEADERS)
    return r.json()

# 使用示例
plans = get_plans()
plan_id = plans["data"][0]["plan"]["id"]

# Epay
pay_url, params = pay_epay(plan_id, "alipay")
print(f"请用户访问：{pay_url}")

# Stripe
# pay_link = pay_stripe(plan_id)
# print(f"请用户访问：{pay_link}")
```

---

## 六、前端接入建议

### 套餐展示页

1. 调用 `GET /subscription/plans` 拉取套餐列表
2. 按 `sort_order` 排序展示
3. 每个套餐显示：标题、副标题、价格、时长
4. 根据用户选择的支付方式，调用对应的下单接口

### 支付方式选择

前端可以检测当前启用的支付方式：
```http
GET /topup
```

响应中包含：
```json
{
  "enable_online_topup": true,   // Epay
  "enable_stripe_topup": true,   // Stripe
  "enable_creem_topup": true     // Creem
}
```

根据返回的布尔值展示对应的支付按钮。

---

## 七、回调通知（服务器端）

### Epay 回调

new-api 已内置处理，无需下游额外开发。

如需自己处理支付结果，可轮询查询用户订阅状态：
```python
import time

def wait_for_subscription(timeout=60):
    for _ in range(timeout):
        result = get_my_subscription()
        subs = result.get("data", {}).get("subscriptions", [])
        if subs:
            return subs[0]["subscription"]
        time.sleep(1)
    return None
```
