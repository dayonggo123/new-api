# 运营 Banner API 文档

> 版本：v1.0  
> 适用：下游 App / 客户端对接  
> 基地址：`https://<your-domain>/api`  
> 认证方式：Header 携带 `Authorization: Bearer <access_token>`

---

## 一、获取当前生效的 Banner 列表

### 接口

```
GET /marketing/banners
```

### 认证

需要用户登录态（`access_token`）。

### 请求示例

```bash
curl -s "https://api.harse.tv/api/marketing/banners" \
  -H "Authorization: Bearer sk-xxxxxxxxxxxxxxxx"
```

### 响应示例

```json
{
  "success": true,
  "message": "",
  "data": {
    "banners": [
      {
        "id": 1,
        "priority": 100,
        "enabled": true,
        "start_at": 1752134400,
        "end_at": 1754726400,
        "max_dismiss_hours": 24,
        "content": {
          "zh": {
            "text": "限时福利！新用户注册立享 7 天 VIP 会员",
            "cta": "立即领取",
            "action_type": "open_price_table",
            "action_payload": ""
          },
          "en": {
            "text": "Limited time offer! New users get 7-day VIP free",
            "cta": "Get it now",
            "action_type": "open_price_table",
            "action_payload": ""
          }
        }
      },
      {
        "id": 2,
        "priority": 50,
        "enabled": true,
        "start_at": 0,
        "end_at": 0,
        "max_dismiss_hours": 72,
        "content": {
          "zh": {
            "text": "邀请好友加入，双方各得 $0.3 额度奖励",
            "cta": "去邀请",
            "action_type": "open_invite",
            "action_payload": ""
          }
        }
      }
    ]
  }
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | Banner 唯一标识 |
| `priority` | int | 优先级，数字越大越靠前展示 |
| `enabled` | bool | 是否启用 |
| `start_at` | int64 | 生效开始时间（秒级时间戳），`0` 表示不限 |
| `end_at` | int64 | 生效结束时间（秒级时间戳），`0` 表示不限 |
| `max_dismiss_hours` | int | 用户手动关闭后，多少小时内不再展示。默认 `24` |
| `content` | object | 按语言编码的内容对象，key 为语言代码（如 `zh`、`en`） |

#### Content 对象结构

| 字段 | 类型 | 说明 |
|------|------|------|
| `text` | string | 主文案（必填） |
| `cta` | string | 按钮文案，如"立即领取" |
| `action_type` | string | 点击行为类型，见下表 |
| `action_payload` | string | 附加参数，视 `action_type` 而定 |

#### action_type 枚举

| 值 | 说明 | payload 示例 |
|----|------|-------------|
| `open_url` | 外部浏览器打开链接 | `https://example.com` |
| `open_price_table` | 打开 App 内价格表 | 空字符串 `""` |
| `open_invite` | 打开邀请返利页 | 空字符串 `""` |
| `open_settings` | 打开设置面板 | `category` 值，如 `"account"` |
| `noop` | 纯展示，无点击行为 | 空字符串 `""` |

---

## 二、客户端展示逻辑建议

### 1. 过滤已关闭的 Banner

客户端本地维护一个 `dismissed_banners` 缓存（key 格式：`banner:{id}:dismissed_at`）。

每次获取列表后：
1. 读取 `dismissed_at` 时间戳
2. 如果当前时间 - `dismissed_at` < `max_dismiss_hours * 3600`，则跳过该 banner
3. 否则（未关闭或已超冷却时间），展示该 banner

### 2. 多语言匹配

按以下优先级匹配 `content`：
1. 精确匹配用户当前语言（如 `zh`）
2. 匹配语言前缀（如用户是 `zh-CN`，fallback 到 `zh`）
3. fallback 到 `en`（如果存在）
4. 取 `content` 中第一个可用语言

### 3. 优先级排序

接口已按 `priority DESC, created_at DESC` 排序，客户端按返回顺序展示即可。

---

## 三、管理员接口（后台使用）

以下接口需要管理员权限（`AdminAuth`），由后台管理页面调用。

### 3.1 获取所有 Banner

```
GET /admin/marketing/banners
Authorization: Bearer <admin_token>
```

响应：
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "priority": 100,
      "enabled": true,
      "start_at": 1752134400,
      "end_at": 1754726400,
      "max_dismiss_hours": 24,
      "content": "{\"zh\":{\"text\":\"...\",\"cta\":\"...\",\"action_type\":\"open_url\",\"action_payload\":\"https://...\"}}",
      "created_at": 1752134400,
      "updated_at": 1752134400
    }
  ]
}
```

> 注意：管理员列表返回的 `content` 是 JSON 字符串，前端需要 `JSON.parse`。

### 3.2 创建 Banner

```
POST /admin/marketing/banners
Authorization: Bearer <admin_token>
Content-Type: application/json
```

请求体：
```json
{
  "priority": 100,
  "enabled": true,
  "start_at": 1752134400,
  "end_at": 1754726400,
  "max_dismiss_hours": 24,
  "content": {
    "zh": {
      "text": "限时福利！新用户注册立享 7 天 VIP 会员",
      "cta": "立即领取",
      "action_type": "open_price_table",
      "action_payload": ""
    },
    "en": {
      "text": "Limited time offer! New users get 7-day VIP free",
      "cta": "Get it now",
      "action_type": "open_price_table",
      "action_payload": ""
    }
  }
}
```

### 3.3 更新 Banner

```
PUT /admin/marketing/banners
Authorization: Bearer <admin_token>
Content-Type: application/json
```

请求体（需包含 `id`）：
```json
{
  "id": 1,
  "priority": 120,
  "enabled": true,
  "start_at": 1752134400,
  "end_at": 1754726400,
  "max_dismiss_hours": 48,
  "content": {
    "zh": {
      "text": "文案已更新",
      "cta": "去看看",
      "action_type": "open_url",
      "action_payload": "https://harse.tv"
    }
  }
}
```

### 3.4 删除 Banner

```
DELETE /admin/marketing/banners/:id
Authorization: Bearer <admin_token>
```

---

## 四、错误响应格式

```json
{
  "success": false,
  "message": "错误描述",
  "data": null
}
```

常见错误码：

| HTTP 状态码 | 说明 |
|------------|------|
| 401 | Token 无效或过期 |
| 403 | 非管理员权限 |
| 400 | 参数错误（如 `content` 必填） |
| 500 | 服务端内部错误 |

---

## 五、快速测试

### 用 cURL 测试用户端接口

```bash
# 1. 先登录获取 access_token（或直接用已有 token）
TOKEN="sk-xxxxxxxx"

# 2. 获取 banner 列表
curl -s "https://api.harse.tv/api/marketing/banners" \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### 用 cURL 测试管理端接口

```bash
ADMIN_TOKEN="sk-admin-xxxx"

# 创建 Banner
curl -s -X POST "https://api.harse.tv/api/admin/marketing/banners" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": 100,
    "enabled": true,
    "start_at": 0,
    "end_at": 0,
    "max_dismiss_hours": 24,
    "content": {
      "zh": {
        "text": "测试 Banner 文案",
        "cta": "点击测试",
        "action_type": "noop",
        "action_payload": ""
      }
    }
  }' | jq .

# 列表
curl -s "https://api.harse.tv/api/admin/marketing/banners" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
```
