# 钱包管理 - 订阅套餐模块 下游对接方案

> 目标：在下游应用中复现 HeHarseCloud 前台「钱包管理 → 订阅套餐」页面的完整功能。
>
> 接口域名：`https://heharse.cloud`

---

## 一、页面功能拆解

你截图中的「订阅套餐」页面由以下几个部分组成：

| 页面元素 | 数据来源接口 | 说明 |
|---------|-------------|------|
| **订阅套餐标签** | 前端静态 | 切换「订阅套餐 / 额度充值」 |
| **我的订阅 · 无生效** | `GET /subscription/self` | 显示当前是否有生效的会员 |
| **优先钱包 ▼** | `GET /subscription/self` + `PUT /subscription/self/preference` | 计费偏好，可切换 |
| **套餐卡片列表** | `GET /subscription/plans` | VIP / SVIP 卡片展示 |
| **立即订阅按钮** | `POST /subscription/epay/pay` 等 | 点击后跳转支付 |

---

## 二、认证方式（重点）

**所有订阅接口必须使用 Access Token 认证，不能用 API Key。**

```http
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
```

- **AccessToken**：后台「个人设置」页面复制的字符串
- **UserId**：后台「个人设置」页面看到的数字用户ID

---

## 三、接口调用流程

### 步骤 1：页面初始化时并行请求

```javascript
// 同时请求套餐列表 + 我的订阅状态
const [plansRes, selfRes] = await Promise.all([
  fetch("https://heharse.cloud/subscription/plans", {
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "New-Api-User": userId
    }
  }).then(r => r.json()),
  
  fetch("https://heharse.cloud/subscription/self", {
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "New-Api-User": userId
    }
  }).then(r => r.json())
]);
```

---

### 步骤 2：渲染「我的订阅」状态栏

**接口：** `GET /api/subscription/self`

**响应结构：**
```json
{
  "success": true,
  "data": {
    "billing_preference": "subscription_first",
    "subscriptions": [],           // 当前生效的订阅
    "all_subscriptions": []        // 所有订阅（含过期）
  }
}
```

**渲染逻辑：**

```javascript
// 判断是否有生效订阅
const activeSubs = selfRes.data.subscriptions;
const hasActive = activeSubs.length > 0;

// 显示文本
const statusText = hasActive 
  ? `${activeSubs[0].plan_name} · 生效中`  // 如 "VIP · 生效中"
  : "无生效";                               // 如截图所示

// 计费偏好
const billingPref = selfRes.data.billing_preference;
// "subscription_first" → 优先订阅
// "wallet_first" → 优先钱包
```

---

### 步骤 3：渲染套餐卡片

**接口：** `GET /api/subscription/plans`

**响应结构：**
```json
{
  "success": true,
  "data": [
    {
      "plan": {
        "id": 2,
        "title": "SVIP",
        "price_amount": 20.90,
        "currency": "USD",
        "duration_value": 1,
        "duration_unit": "month",
        "total_amount": 0,
        "reset_strategy": "monthly",
        "upgrade_group": "svip",
        "enabled": true
      }
    },
    {
      "plan": {
        "id": 1,
        "title": "VIP",
        "price_amount": 12.90,
        "currency": "USD",
        "duration_value": 1,
        "duration_unit": "month",
        "total_amount": 0,
        "reset_strategy": "monthly",
        "upgrade_group": "vip",
        "enabled": true
      }
    }
  ]
}
```

**卡片渲染字段映射：**

| 页面显示 | 接口字段 | 示例 |
|---------|---------|------|
| 套餐名 | `plan.title` | SVIP / VIP |
| 价格 | `$${plan.price_amount}` | $20.90 / $12.90 |
| 有效期 | `有效期: ${duration_value}${duration_unit === 'month' ? '个月' : ''}` | 有效期: 1个月 |
| 额度重置 | `额度重置: ${reset_strategy === 'monthly' ? '每月' : ''}` | 额度重置: 每月 |
| 总额度 | `总额度: ${plan.total_amount === 0 ? '不限' : plan.total_amount}` | 总额度: 不限 |
| 升级分组 | `升级分组: ${plan.upgrade_group}` | 升级分组: svip |

**注意：** 你的套餐 `total_amount` 都是 0，表示**不限额度纯会员模式**。

---

### 步骤 4：点击「立即订阅」下单

根据用户选择的支付方式，调用对应接口：

#### A. Epay（支付宝）

```http
POST /api/subscription/epay/pay
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
Content-Type: application/json

{
  "plan_id": 1,
  "payment_method": "alipay"
}
```

**响应：**
```json
{
  "success": true,
  "data": "https://pay.example.com/pay?trade_no=SUB-xxxx"
}
```

**前端操作：** `window.open(response.data, '_blank')` 或 `location.href = response.data`

#### B. Epay（微信）

```json
{
  "plan_id": 1,
  "payment_method": "wxpay"
}
```

#### C. Stripe

```http
POST /api/subscription/stripe/pay

{
  "plan_id": 1
}
```

#### D. Creem

```http
POST /api/subscription/creem/pay

{
  "plan_id": 1
}
```

---

### 步骤 5：修改计费偏好（可选）

如果用户点击「优先钱包 ▼」想切换计费方式：

```http
PUT /api/subscription/self/preference
Authorization: Bearer {AccessToken}
New-Api-User: {UserId}
Content-Type: application/json

{
  "billing_preference": "wallet_first"
}
```

**可选值：**
- `"subscription_first"` — 优先使用订阅额度
- `"wallet_first"` — 优先使用钱包余额

---

## 四、完整 React/Vue 伪代码示例

```javascript
// ===== 数据获取 =====
async function loadSubscriptionData() {
  const headers = {
    "Authorization": `Bearer ${accessToken}`,
    "New-Api-User": userId
  };
  
  const [plansRes, selfRes] = await Promise.all([
    fetch("/api/subscription/plans", { headers }).then(r => r.json()),
    fetch("/api/subscription/self", { headers }).then(r => r.json())
  ]);
  
  return {
    plans: plansRes.data || [],
    subscriptions: selfRes.data?.subscriptions || [],
    allSubscriptions: selfRes.data?.all_subscriptions || [],
    billingPreference: selfRes.data?.billing_preference || "subscription_first"
  };
}

// ===== 页面渲染 =====
function SubscriptionPage() {
  const [plans, setPlans] = useState([]);
  const [subscriptions, setSubscriptions] = useState([]);
  const [billingPref, setBillingPref] = useState("subscription_first");
  
  useEffect(() => {
    loadSubscriptionData().then(data => {
      setPlans(data.plans);
      setSubscriptions(data.subscriptions);
      setBillingPref(data.billingPreference);
    });
  }, []);
  
  const hasActiveSub = subscriptions.length > 0;
  const activeSubName = hasActiveSub ? subscriptions[0].plan_name : null;
  
  return (
    <div className="subscription-page">
      {/* 我的订阅状态 */}
      <div className="sub-status">
        <span>我的订阅</span>
        <span className={hasActiveSub ? "active" : "inactive"}>
          {hasActiveSub ? `${activeSubName} · 生效中` : "无生效"}
        </span>
        <select value={billingPref} onChange={handlePrefChange}>
          <option value="subscription_first">优先订阅</option>
          <option value="wallet_first">优先钱包</option>
        </select>
      </div>
      
      {/* 套餐卡片列表 */}
      <div className="plans-grid">
        {plans.map(({ plan }) => (
          <div key={plan.id} className="plan-card">
            <h3>{plan.title}</h3>
            <div className="price">${plan.price_amount}</div>
            <ul className="features">
              <li>有效期: {plan.duration_value}个月</li>
              <li>额度重置: {plan.reset_strategy === 'monthly' ? '每月' : plan.reset_strategy}</li>
              <li>总额度: {plan.total_amount === 0 ? '不限' : plan.total_amount}</li>
              <li>升级分组: {plan.upgrade_group}</li>
            </ul>
            <button onClick={() => handleSubscribe(plan.id)}>
              立即订阅
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}

// ===== 支付下单 =====
async function handleSubscribe(planId) {
  const res = await fetch("/api/subscription/epay/pay", {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "New-Api-User": userId,
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      plan_id: planId,
      payment_method: "alipay"  // 或 wxpay
    })
  }).then(r => r.json());
  
  if (res.success && res.data) {
    window.open(res.data, '_blank');  // 跳转支付
  } else {
    alert(res.message || "下单失败");
  }
}

// ===== 切换计费偏好 =====
async function handlePrefChange(e) {
  const newPref = e.target.value;
  const res = await fetch("/api/subscription/self/preference", {
    method: "PUT",
    headers: {
      "Authorization": `Bearer ${accessToken}`,
      "New-Api-User": userId,
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ billing_preference: newPref })
  }).then(r => r.json());
  
  if (res.success) {
    setBillingPref(newPref);
  }
}
```

---

## 五、支付完成后的处理

用户支付完成后，需要刷新订阅状态：

```javascript
// 支付页返回后，或定时轮询
async function checkSubscriptionStatus() {
  const res = await fetch("/api/subscription/self", {
    headers: { "Authorization": `Bearer ${accessToken}`, "New-Api-User": userId }
  }).then(r => r.json());
  
  const hasActive = (res.data?.subscriptions || []).length > 0;
  if (hasActive) {
    // 更新用户 group 字段（vip / svip）
    // 刷新页面状态
    window.location.reload();
  }
}
```

---

## 六、常见问题

### Q: 为什么获取套餐返回空数组？

A: 检查三点：
1. 用的是 **Access Token** 不是 API Key
2. 带了 `New-Api-User` header
3. 后台「订阅管理」里套餐是**启用**状态

### Q: `total_amount: 0` 是什么意思？

A: **0 表示不限额度**。这是纯会员费模式，用户购买后只获得会员身份（不同分组享受不同价格），不消耗额度。

### Q: 支付成功后用户没变成会员？

A: 检查 Epay 回调地址是否可外网访问：`https://heharse.cloud/subscription/epay/notify`

### Q: 用户分组怎么用？

A: 购买后用户 `group` 字段变成 `vip` 或 `svip`，下游可用这个字段控制模型价格/权限。

---

## 七、接口速查表

| 功能 | 方法 | 路径 | 认证 |
|------|------|------|------|
| 获取套餐列表 | GET | `/api/subscription/plans` | Access Token + New-Api-User |
| 获取我的订阅 | GET | `/api/subscription/self` | Access Token + New-Api-User |
| 修改计费偏好 | PUT | `/api/subscription/self/preference` | Access Token + New-Api-User |
| Epay 支付下单 | POST | `/api/subscription/epay/pay` | Access Token + New-Api-User |
| Stripe 支付下单 | POST | `/api/subscription/stripe/pay` | Access Token + New-Api-User |
| Creem 支付下单 | POST | `/api/subscription/creem/pay` | Access Token + New-Api-User |
