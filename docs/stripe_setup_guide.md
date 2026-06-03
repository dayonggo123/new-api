# Stripe 支付配置完整步骤

## 第一步：进入 Stripe 产品页面

1. 在你当前的 Stripe 后台页面，点击左侧菜单（如果没有看到菜单，点击左上角 ☰ 展开）
2. 找到 **"产品目录"** 或 **"Products"**，点击进入

或者直接访问这个链接：
```
https://dashboard.stripe.com/test/products
```

---

## 第二步：创建产品

1. 点击右上角 **"添加产品"**（Add Product）
2. 填写产品信息：
   - **产品名称**：`VIP 会员`（或任意名称）
   - **描述**：`月度 VIP 订阅`（可选）

---

## 第三步：添加价格（关键步骤）

在产品编辑页面，找到 **"价格信息"** 部分：

| 选项 | 填写值 |
|------|--------|
| 定价模式 | 标准定价 |
| 价格 | `12.90` |
| 货币 | `USD` |
| 计费方式 | **循环计费**（Recurring） |
| 间隔 | `每月`（Month） |

点击 **保存**。

> 同样的步骤再创建 SVIP 的价格（$20.90）。

---

## 第四步：复制 Price ID

保存后，你会看到刚刚创建的价格，格式如下：

```
price_1TUkqxCmACxwPcWozxmrBgB0e0aXxv
```

点击复制这个 ID。

---

## 第五步：填到 new-api 后台

登录 new-api 后台 → **订阅管理** → 点击 **VIP** 套餐的编辑：

找到字段 **"Stripe 商品价格 ID"**，粘贴刚才复制的 Price ID。

对 SVIP 套餐重复同样操作（用 SVIP 对应的 Price ID）。

---

## 第六步：配置 Stripe API 密钥

在 Stripe 后台：
1. 点击左侧 **"开发人员"** → **"API 密钥"**
2. 复制 **密钥**（Secret Key，以 `sk_test_` 开头的是测试密钥）
3. 粘贴到 new-api 后台 → **系统设置** → **支付设置** → **Stripe API Secret**

---

## 第七步：配置 Webhook（支付回调）

1. Stripe 后台 → **开发人员** → **Webhook**
2. 点击 **"添加端点"**
3. 端点 URL 填：
   ```
   https://heharse.cloud/api/subscription/stripe/webhook
   ```
4. 选择监听事件：
   - `checkout.session.completed`
5. 保存后，复制 **签名密钥**（Webhook Secret，以 `whsec_` 开头）
6. 粘贴到 new-api 后台 → **系统设置** → **支付设置** → **Stripe Webhook Secret**

---

## 常见问题

**Q：Price ID 在哪里看？**  
A：在产品详情页 → 价格列表 → 点击展开价格 → 看到 `price_xxxxx` 就是。

**Q：沙盒和真实账户有什么区别？**  
A：你现在在沙盒（测试环境），Price ID 是测试的。正式上线前要切换到真实账户，重新创建产品和价格。

**Q：可以一个产品填多个 Price ID 吗？**  
A：不行。new-api 每个套餐只能填一个 Price ID，VIP 和 SVIP 各填各的。

---

如果还是找不到，把 Stripe 后台当前页面的完整截图发我，我指给你看按钮在哪里。
