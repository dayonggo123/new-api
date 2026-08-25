# Referral 邀请码接口文档

## 概述

社媒分享邀请码服务端接口。每个用户拥有唯一的 6 位分享码，可用于注册邀请、来源追踪。

---

## 1. 获取当前用户邀请码

**接口**：`GET /api/referral/my-code`

**鉴权**：需要登录

**说明**：返回当前用户的邀请码、短链接、二维码。若用户尚无邀请码，自动生成。

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "user_id": 42,
    "code": "A1B2C3",
    "short_link": "https://api.lk888.ai/r/A1B2C3",
    "qr_url": "https://api.lk888.ai/api/public/qr?data=https%3A%2F%2Fapi.lk888.ai%2Fr%2FA1B2C3",
    "expires_at": null,
    "max_uses": 0,
    "used_count": 0,
    "created_time": 1758739200,
    "updated_time": 1758739200
  }
}
```

---

## 2. 主动生成/获取邀请码

**接口**：`POST /api/referral/generate`

**鉴权**：需要登录

**说明**：用于应用内分享弹窗。按用户维度幂等：同一用户始终返回同一个邀请码。`source` / `content_id` 仅用于分享来源统计，不影响码本身。

**请求参数**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `source` | string | 否 | 分享来源，如 `image` / `template` / `video` |
| `content_id` | string | 否 | 分享内容 ID |

**响应示例**：同 `GET /api/referral/my-code`

---

## 3. 校验邀请码

**接口**：`GET /api/referral/validate?code=A1B2C3`

**鉴权**：公开

**说明**：供落地页和注册页使用，判断邀请码是否有效。

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "valid": true,
    "inviter_id": 42,
    "inviter_name": "username",
    "reward_preview": {
      "register_bonus": 500000,
      "topup_rebate_ratio": 0
    }
  }
}
```

---

## 数据模型

### referral_codes

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 主键 |
| `user_id` | int | 用户 ID，唯一 |
| `code` | string(16) | 6 位邀请码，唯一 |
| `short_link` | string(255) | 短链接 |
| `qr_url` | string(255) | 二维码链接 |
| `expires_at` | int64 | 过期时间戳，可选 |
| `max_uses` | int | 最大使用次数，0 为不限 |
| `used_count` | int | 已使用次数 |
| `created_time` | int64 | 创建时间 |
| `updated_time` | int64 | 更新时间 |

---

## 注册接入

注册时通过 URL 参数 `?ref=A1B2C3` 传递邀请码，服务端在注册流程中调用 `service.ValidateReferralCode(code)` 获取邀请人 ID，写入 `users.inviter_id`，并发放注册奖励。

> 当前 MVP 版本仅提供邀请码生成与校验接口，注册链路改造在后续迭代中完成。
