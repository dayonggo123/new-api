# Referral 邀请码 下游对接文档

## 功能概述

用户拥有一条唯一的 6 位邀请码。分享短链给好友，好友通过短链注册后，二人建立邀请关系，并自动发放注册奖励。

---

## 1. 获取当前用户邀请码

**接口**：`GET /api/user/referral/my-code`

**鉴权**：登录用户（UserAuth）

**说明**：返回当前用户的邀请码、短链接、二维码。若不存在，自动生成。

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

**接口**：`POST /api/user/referral/generate`

**鉴权**：登录用户（UserAuth）

**说明**：用于应用内分享弹窗。幂等：同一用户始终返回同一个邀请码。`source` / `content_id` 仅用于统计，不影响码。

**请求体**：

```json
{
  "source": "image",
  "content_id": "tpl_123"
}
```

**响应示例**：同 `GET /api/user/referral/my-code`

---

## 3. 校验邀请码

**接口**：`GET /api/user/referral/validate?code=A1B2C3`

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

## 4. 短链跳转

**接口**：`GET /r/:code`

**说明**：分享出去的短链。后端校验 code 有效后，把 code 写入 session，并重定向到注册/落地页。

**示例**：

```text
https://api.lk888.ai/r/A1B2C3
```

**跳转目标**：

```text
https://api.lk888.ai/register?ref=A1B2C3
```

> 前端落地页/注册页从 URL 读取 `ref` 参数展示即可，无需额外处理。session 中已保存 `ref_code`。

---

## 5. 注册时携带邀请码

### 5.1 普通注册

**接口**：`POST /api/user/register`

邀请码可以通过以下任一方式传入：

1. 请求头：`X-Referral-Code: A1B2C3`
2. 请求体：

```json
{
  "username": "newuser",
  "password": "xxx",
  "aff_code": "A1B2C3"
}
```

3. 通过 `/r/:code` 访问后，session 中已有 `ref_code`，注册时自动携带。

> 优先级：`aff_code` 请求体 > `X-Referral-Code` 请求头 > session `ref_code`

### 5.2 OAuth 注册

OAuth 授权前调用 `/api/oauth/state?ref=A1B2C3`，服务端会把 ref 写入 session。注册回调时自动使用。

```text
GET /api/oauth/state?ref=A1B2C3
```

---

## 6. 邀请关系与奖励

注册成功后，服务端自动完成：

1. 校验邀请码有效性
2. 检查自邀（邀请人不能等于注册用户）
3. 写入 `referral_relationships` 表
4. `referral_codes.used_count` 自增
5. 按系统配置给邀请人和被邀请人发放额度奖励（复用 `QuotaForInvitee` / `QuotaForInviter`）

---

## 7. 数据表

### 7.1 referral_codes

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 主键 |
| user_id | int | 用户 ID，唯一 |
| code | string(16) | 6 位邀请码，唯一 |
| short_link | string(255) | 短链接 |
| qr_url | string(255) | 二维码链接 |
| expires_at | int64 | 过期时间，可选 |
| max_uses | int | 最大使用次数，0 为不限 |
| used_count | int | 已使用次数 |
| created_time | int64 | 创建时间 |
| updated_time | int64 | 更新时间 |

### 7.2 referral_relationships

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 主键 |
| referral_code_id | int | 邀请码 ID |
| inviter_id | int | 邀请人 ID |
| invitee_id | int | 被邀请人 ID |
| code | string(16) | 邀请码 |
| source | string(32) | 分享来源 |
| content_id | string(64) | 内容 ID |
| ip | string(64) | 注册 IP |
| device_fingerprint | string(128) | 设备指纹 |
| created_time | int64 | 创建时间 |

---

## 8. 防刷规则

当前版本已实现：

- 自邀拦截：邀请人不能邀请自己
- 邀请码过期/使用次数上限校验
- 无效码不建立关系

待完善（建议下一迭代）：

- 同一 IP 注册次数限制
- 同一设备指纹注册次数限制
- 同一邮箱/手机号重复注册拦截

---

## 9. 接口清单汇总

| 方法 | 路径 | 鉴权 | 用途 |
|------|------|------|------|
| GET | `/api/user/referral/my-code` | UserAuth | 获取我的邀请码 |
| POST | `/api/user/referral/generate` | UserAuth | 主动生成/获取邀请码 |
| GET | `/api/user/referral/validate` | 公开 | 校验邀请码 |
| GET | `/r/:code` | 公开 | 短链跳转并记录 session |
| POST | `/api/user/register` | 公开 | 普通注册（支持携带邀请码） |
| GET | `/api/oauth/state` | 公开 | OAuth 授权前写入邀请码 |

---

## 10. 前端对接建议

### 分享弹窗

1. 调用 `POST /api/user/referral/generate`，拿到 `short_link` 和 `qr_url`
2. 展示复制链接按钮和二维码
3. 分享时可选择来源（如 image/template/video），传入 `source` / `content_id`

### 注册页

1. 从 URL 读取 `?ref=A1B2C3`
2. 调用 `GET /api/user/referral/validate?code=A1B2C3` 校验并展示邀请人信息
3. 注册时通过 `X-Referral-Code` 请求头或 `aff_code` 请求体携带邀请码
