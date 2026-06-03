# 订阅接口升级通知

> 升级时间：2026-06-02
> 影响范围：`/api/subscription/plans`、`/api/subscription/group-discount`
> 优先级：高（前端需在下次发版前适配）

---

## 一、变更总览

| 接口 | 变更类型 | 说明 |
|------|----------|------|
| `GET /api/subscription/plans` | **返回格式升级** | 新增折扣相关字段（原价、折扣价、分组倍率、折扣百分比） |
| `GET /api/subscription/group-discount` | **接口修复** | 从纯文本改为返回标准 JSON，包含用户当前分组折扣信息 |

**核心改动点：**
- 套餐价格不再只有单一 `price_amount`，而是拆分为 `original_price` + `discounted_price`
- 折扣由后端根据用户所属分组自动计算，**前端无需传任何折扣参数**
- 支付下单接口（`epay/yizhifu/stripe/creem`）已移除 `discount_code` 字段

---

## 二、接口详情

### 2.1 GET /api/subscription/plans（已升级）

**变更前响应：**
```json
{
  "message": "success",
  "data": [
    {
      "plan": {
        "id": 1,
        "title": "月度套餐",
        "price_amount": 99.99,
        ...
      }
    }
  ]
}
```

**变更后响应：**
```json
{
  "message": "success",
  "data": [
    {
      "plan": {
        "id": 1,
        "title": "VIP",
        "price_amount": 6.99,
        "duration_days": 30,
        "enabled": true,
        ...
      },
      "original_price": 6.99,
      "discounted_price": 4.89,
      "group_ratio": 0.7,
      "discount_percent": 30.0
    }
  ]
}
```

**新增字段说明：**

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `original_price` | float | 套餐原价 | `6.99` |
| `discounted_price` | float | 按用户分组倍率计算后的折扣价 | `4.89` |
| `group_ratio` | float | 分组倍率（用于前端模型计价） | `0.7` |
| `discount_percent` | float | 折扣百分比（用于展示折扣标签） | `30.0` |

**前端适配建议：**
1. 展示价格时优先使用 `discounted_price`
2. 当 `discount_percent > 0` 时，显示原价划线 + 折扣标签（如 "VIP 专享 7 折"）
3. `group_ratio` 可用于前端其他按量计费的模型价格展示

---

### 2.2 GET /api/subscription/group-discount（已修复）

**变更前：** 返回纯文本 `"New API"`（异常）

**变更后响应：**
```json
{
  "message": "success",
  "data": {
    "user_group": "svip",
    "group_ratio": 0.5,
    "discount_percent": 50.0
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `user_group` | string | 用户当前所属分组标识 | `"svip"`、`"vip"`、`"default"` |
| `group_ratio` | float | 分组倍率 | `0.5` |
| `discount_percent` | float | 折扣百分比 | `50.0` |

**前端适配建议：**
- 在套餐页顶部或个人中心调用此接口，展示当前用户的分组折扣信息
- 可作为全局状态缓存，减少重复请求

---

## 三、后台分组倍率配置

当前后台已配置的分组倍率如下：

| 分组 | 倍率 | 折扣 | 说明 |
|------|------|------|------|
| `svip` | 0.5 | 5 折 | 最高等级 |
| `vip` | 0.7 | 7 折 | 中间等级 |
| `default` | 1.0 | 无折扣 | 默认分组 |

> 管理员可在后台「分组管理」中调整倍率，调整后立即生效。

---

## 四、支付下单接口变更

**以下四个接口已移除 `discount_code` 参数：**

| 接口 | 当前请求体 |
|------|-----------|
| `POST /api/subscription/epay/pay` | `{ "plan_id": 1, "payment_method": "alipay" }` |
| `POST /api/subscription/yizhifu/pay` | `{ "plan_id": 1, "payment_method": "alipay" }` |
| `POST /api/subscription/stripe/pay` | `{ "plan_id": 1 }` |
| `POST /api/subscription/creem/pay` | `{ "plan_id": 1 }` |

**前端需要做的：**
- 删除所有折扣码输入框、验证按钮及相关 UI
- 用户选择套餐后直接调支付接口即可，后端自动按分组倍率计算最终价格

---

## 五、前端接入示例

### 5.1 套餐列表页展示

```javascript
// 获取套餐列表
const res = await fetch('/api/subscription/plans');
const { data } = await res.json();

data.forEach(item => {
  const { plan, original_price, discounted_price, discount_percent } = item;

  if (discount_percent > 0) {
    // 有折扣：展示折扣价 + 原价划线 + 折扣标签
    renderPlanCard({
      title: plan.title,
      price: discounted_price,          // 优先展示折扣价
      originalPrice: original_price,    // 划线价
      badge: `${discount_percent}折`,   // 折扣标签
    });
  } else {
    // 无折扣：正常展示
    renderPlanCard({
      title: plan.title,
      price: original_price,
    });
  }
});
```

### 5.2 用户分组折扣横幅

```javascript
// 在个人中心或套餐页顶部展示
const res = await fetch('/api/subscription/group-discount');
const { data } = await res.json();

if (data.discount_percent > 0) {
  showBanner(
    `您当前为 ${data.user_group.toUpperCase()} 用户，订阅享 ${data.discount_percent} 折优惠`
  );
}
```

### 5.3 支付下单（无折扣码）

```javascript
// 直接下单，无需传折扣码
const res = await fetch('/api/subscription/epay/pay', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    plan_id: selectedPlan.id,
    payment_method: 'alipay'
  })
});
```

---

## 六、订单相关字段（Webhook / 订单查询）

创建订单后，订单对象包含以下折扣相关字段：

```json
{
  "original_amount": 6.99,
  "discount_amount": 2.10,
  "money": 4.89
}
```

| 字段 | 说明 |
|------|------|
| `original_amount` | 套餐原价 |
| `discount_amount` | 减免金额 |
| `money` | 实际需支付金额 |

---

## 七、常见问题

**Q1：如果用户分组变了，已创建的订单价格会变吗？**
A：不会。订单创建时即按当时的分组倍率锁定价格，后续分组变更不影响已创建订单。

**Q2：`group_ratio` 可能大于 1 吗？**
A：理论上可能（如分组倍率 1.2 表示溢价），但当前后台配置均为 <= 1.0。前端按通用逻辑处理即可。

**Q3：`discount_percent` 为 0 时前端怎么展示？**
A：按无折扣处理，只展示一个价格即可，不需要划线价和折扣标签。

**Q4：未登录用户能获取套餐列表吗？**
A：可以，但未登录用户的 `group_ratio` 为 `1.0`，`discount_percent` 为 `0`，即按原价展示。

---

## 八、排期建议

| 阶段 | 事项 | 建议时间 |
|------|------|----------|
| 1 | 套餐列表页适配新字段 | 本周内 |
| 2 | 移除折扣码相关 UI | 本周内 |
| 3 | 添加用户分组折扣横幅 | 下周 |
| 4 | 联调测试 | 发版前 |

---

如有疑问请联系后端团队。
