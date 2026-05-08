# new-api 积分/签到/消息系统 — 下游对接文档

> 版本：v1.0 | 日期：2026-05-08

---

## 一、基本信息

| 项目 | 说明 |
|------|------|
| **接口基地址** | `https://heharse.cloud/api` |
| **认证方式** | Access Token / Cookie Session + `New-Api-User` Header |
| **请求格式** | `application/json` |
| **响应格式** | 统一 JSON：`{ success: boolean, message: string, data: any }` |

---

## 二、认证机制

### 2.1 方式一：WebView 回调登录（推荐用于移动端/App）

适用于 ewapi 等下游应用内嵌 WebView 的场景。

**流程**：
1. ewapi 打开 WebView，访问：
   ```
   https://heharse.cloud/login?callback=ewapi://auth/callback
   ```
2. 用户在 WebView 中完成登录
3. 登录成功后，自动 302 重定向到：
   ```
   ewapi://auth/callback?token=xxx&user_id=1
   ```
4. ewapi 拦截该 URL，提取 `token` 和 `user_id`
5. 后续 API 请求使用 Access Token

**回调参数说明**：
| 参数 | 说明 |
|------|------|
| `token` | Access Token，后续请求放在 `Authorization: Bearer <token>` Header 中 |
| `user_id` | 用户 ID，后续请求放在 `New-Api-User: <user_id>` Header 中 |

**支持的 callback 前缀**：
- `ewapi://`（推荐）
- `http://localhost`
- `https://localhost`

**注意**：如果不传 `callback` 参数，则走普通 Web 登录流程（返回 JSON + Set-Cookie）。

---

### 2.2 方式二：Cookie Session 登录（推荐用于 Web 端）

```http
POST /api/user/login
Content-Type: application/json

{
  "username": "用户账号",
  "password": "密码"
}
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "dayonggo",
    "display_name": "Root User",
    "role": 100,
    "group": "default",
    "status": 1
  }
}
```

> 登录成功后，服务端通过 **Set-Cookie** 返回 session，后续请求需携带该 cookie。

---

### 2.3 后续请求认证

所有需要登录的接口，必须携带：

**方式 A：Access Token（WebView 回调后用此方式）**
```http
GET /api/user/points
Authorization: Bearer <token>
New-Api-User: <user_id>
```

**方式 B：Cookie Session（Web 登录后用此方式）**
```http
GET /api/user/points
Cookie: session=xxx
New-Api-User: 1
```

---

## 三、接口清单

### 3.1 用户积分 & 签到

#### GET /api/user/points
获取当前用户积分和签到状态

**认证**：需要登录

**请求参数**：无

**响应示例**：
```json
{
  "success": true,
  "data": {
    "total_points": 150,
    "consecutive_days": 3,
    "last_signin_date": 1778169600,
    "today_signed": false,
    "next_signin_points": 12
  }
}
```

**字段说明**：
| 字段 | 类型 | 说明 |
|------|------|------|
| total_points | int | 当前总积分 |
| consecutive_days | int | 连续签到天数 |
| last_signin_date | int64 | 上次签到时间（Unix 时间戳） |
| today_signed | bool | 今日是否已签到 |
| next_signin_points | int | 下次签到可获得积分 |

---

#### POST /api/user/signin
执行签到

**认证**：需要登录

**请求参数**：无

**响应示例**（成功）：
```json
{
  "success": true,
  "data": {
    "points_earned": 12,
    "bonus_points": 2,
    "total_points": 162,
    "consecutive_days": 4
  }
}
```

**响应示例**（失败）：
```json
{
  "success": false,
  "message": "今日已签到"
}
```

**签到规则**：
- 基础积分：10 分/天
- 连续签到奖励：第 2 天+1，第 3 天+2，第 4-6 天+2，第 7 天+5
- 断签后连续天数归零

---

#### GET /api/user/signin-history
获取签到历史

**认证**：需要登录

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应示例**：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "signin_date": 1778169600,
        "points": 10,
        "bonus_points": 2,
        "total_points_after": 162
      }
    ],
    "total": 30,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 3.2 提示词解锁

#### POST /api/user/unlock-prompt
解锁付费提示词，扣减积分

**认证**：需要登录

**请求体**：
```json
{
  "prompt_id": 123
}
```

**响应示例**（成功）：
```json
{
  "success": true,
  "data": {
    "prompt_id": 123,
    "cost": 50,
    "remaining_points": 100
  }
}
```

**响应示例**（失败）：
```json
{ "success": false, "message": "积分不足，需要 50 积分" }
{ "success": false, "message": "已解锁该提示词" }
{ "success": false, "message": "该提示词免费，无需解锁" }
```

**业务规则**：
- 同一提示词只能解锁一次（幂等）
- 免费提示词（`is_premium=false`）无需解锁
- 积分不足时返回明确错误

---

#### GET /api/user/unlocked-prompts
获取已解锁提示词列表

**认证**：需要登录

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应示例**：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "prompt_id": 123,
        "title": "提示词标题",
        "cover_image_url": "https://...",
        "cost": 50,
        "unlocked_at": 1778210493
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 3.3 消息通知

#### GET /api/notifications
分页获取消息列表（包含全员广播消息）

**认证**：需要登录

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应示例**：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "user_id": 0,
        "title": "系统维护通知",
        "content": "系统将于今晚进行维护...",
        "type": "system",
        "is_read": false,
        "action_url": "",
        "created_time": 1778210493
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

> **注意**：`user_id=0` 表示全员广播消息，所有用户可见。

---

#### GET /api/notifications/unread-count
获取未读消息数

**认证**：需要登录

**响应示例**：
```json
{
  "success": true,
  "data": {
    "unread_count": 5
  }
}
```

---

#### POST /api/notifications/:id/read
标记单条消息已读

**认证**：需要登录

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | int | 消息 ID |

**响应示例**：
```json
{ "success": true }
```

---

#### POST /api/notifications/read-all
标记全部消息已读

**认证**：需要登录

**响应示例**：
```json
{
  "success": true,
  "data": {
    "marked_count": 5
  }
}
```

---

### 3.4 公开提示词（字段扩展）

#### GET /api/public/prompts
获取公开提示词列表（**已扩展 `is_premium` 和 `unlock_cost` 字段**）

**认证**：无需登录

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category_id | int | 否 | 分类筛选 |
| keyword | string | 否 | 关键词搜索 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应示例**（单条提示词）：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 75,
        "title": "提示词标题",
        "content": "提示词内容...",
        "cover_image_url": "https://...",
        "category_name": "分类名",
        "is_premium": true,
        "unlock_cost": 50
      }
    ],
    "total": 72
  }
}
```

**新增字段说明**：
| 字段 | 类型 | 说明 |
|------|------|------|
| is_premium | bool | 是否为付费提示词 |
| unlock_cost | int | 解锁所需积分（`is_premium=true` 时有效） |

---

### 3.5 管理员接口（需要 Admin 权限）

#### POST /api/admin/notifications
发布消息通知

**认证**：需要 Admin 权限

**请求体**：
```json
{
  "title": "系统维护通知",
  "content": "系统将于今晚进行维护...",
  "type": "system",
  "target_type": "all",
  "target_users": [],
  "target_group": "",
  "action_url": ""
}
```

**target_type 说明**：
| 值 | 说明 |
|----|------|
| all | 全员广播（user_id=0） |
| users | 指定用户（需传 target_users 数组） |
| group | 指定用户组（需传 target_group） |

**响应示例**：
```json
{
  "success": true,
  "message": "发送成功"
}
```

---

#### GET /api/admin/notifications
获取所有通知列表（管理端）

**认证**：需要 Admin 权限

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 按类型筛选 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

---

#### POST /api/user/points/adjust
手动调整用户积分

**认证**：需要 Admin 权限

**请求体**：
```json
{
  "user_id": 1,
  "amount": 100,
  "description": "活动补偿"
}
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "user_id": 1,
    "new_total": 110,
    "adjust_amount": 100
  }
}
```

> `amount` 可为负数（扣除积分），但调整后总积分不能为负数。

---

#### GET /api/user/points/transactions
查询用户积分流水

**认证**：需要 Admin 权限

**请求参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | int | 否 | 用户 ID，不传查全部 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

---

#### GET /api/user/signin/stats
签到统计

**认证**：需要 Admin 权限

**响应示例**：
```json
{
  "success": true,
  "data": {
    "today_signin_count": 150,
    "yesterday_signin_count": 145,
    "consecutive_distribution": [
      { "days": 1, "count": 80 },
      { "days": 3, "count": 40 },
      { "days": 7, "count": 20 }
    ]
  }
}
```

---

## 四、错误码说明

| HTTP 状态 | 错误消息 | 说明 |
|----------|---------|------|
| 200 | `success: false` | 业务逻辑错误（积分不足、已签到等） |
| 401 | `Unauthorized, not logged in` | 未登录或 cookie 过期 |
| 401 | `Unauthorized, New-Api-User header not provided` | 缺少 `New-Api-User` Header |
| 403 | `Access denied` | 权限不足（非 Admin 访问 Admin 接口） |
| 404 | - | 路由不存在 |
| 409 | `今日已签到` | 重复签到 |
| 409 | `已解锁该提示词` | 重复解锁 |

---

## 五、前端集成建议

### 5.1 推荐接入流程（WebView 回调方式）

```
1. 用户访问 ewapi
2. 检测本地是否有 Access Token
3. 无 token → 打开 WebView
         → 访问 https://heharse.cloud/login?callback=ewapi://auth/callback
         → 用户完成登录
         → WebView 重定向到 ewapi://auth/callback?token=xxx&user_id=1
         → ewapi 拦截 URL，保存 token 和 user_id
         → 关闭 WebView
4. 有 token → 调用 /api/user/points 显示积分和签到按钮
5. 浏览提示词列表 → is_premium=true 的显示"解锁"按钮
6. 点击解锁 → 调用 /api/user/unlock-prompt → 刷新积分和解锁状态
```

### 5.2 WebView 登录代码示例（移动端）

```typescript
// 打开 WebView 登录
const loginUrl = 'https://heharse.cloud/login?callback=ewapi://auth/callback';

// iOS (WKWebView)
// 拦截 URL 导航
cfunc webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
    if let url = navigationAction.request.url, url.scheme == "ewapi" {
        let components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        let token = components?.queryItems?.first(where: { $0.name == "token" })?.value
        let userId = components?.queryItems?.first(where: { $0.name == "user_id" })?.value
        // 保存 token 和 user_id
        // 关闭 WebView
        decisionHandler(.cancel)
        return
    }
    decisionHandler(.allow)
}

// Android (WebViewClient)
@Override
public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
    Uri uri = request.getUrl();
    if ("ewapi".equals(uri.getScheme())) {
        String token = uri.getQueryParameter("token");
        String userId = uri.getQueryParameter("user_id");
        // 保存 token 和 user_id
        // 关闭 WebView
        return true;
    }
    return false;
}
```

### 5.2 签到按钮状态

| 状态 | 表现 |
|------|------|
| `today_signed = false` | 显示"签到"按钮，可点击 |
| `today_signed = true` | 显示"已签到 ✓"，禁用按钮，显示"连续 X 天" |

### 5.3 提示词卡片状态

```
if (prompt.is_premium) {
  if (已解锁) {
    显示"已解锁"，可正常使用
  } else {
    显示"🔒 解锁（需 X 积分）"
    点击 → 调用 unlockPrompt → 成功则刷新
  }
} else {
  免费提示词，直接可用
}
```

### 5.4 消息红点提示

```
轮询 GET /api/notifications/unread-count（每 30 秒）
if (unread_count > 0) {
  显示红点角标
}
```

---

## 六、SDK 使用（TypeScript）

已提供封装好的 SDK：`sdk/ewapi-client.ts`

```typescript
import { EwapiClient } from './sdk/ewapi-client';

// ============ WebView 回调登录（移动端/App）============
// 1. 打开 WebView，访问：https://heharse.cloud/login?callback=ewapi://auth/callback
// 2. 用户登录后，WebView 会重定向到 ewapi://auth/callback?token=xxx&user_id=1
// 3. 拦截 URL 提取 token 和 user_id

const client = new EwapiClient('https://heharse.cloud');

// 设置从 WebView 回调中获取的 token
client.setToken('从 WebView 回调中获取的 token');

// 后续请求会自动带上 Authorization: Bearer <token>
const points = await client.getUserPoints();
console.log(points.total_points, points.today_signed);

// ============ Cookie 登录（Web 端）============
// const client = new EwapiClient('https://heharse.cloud');
// await client.login('username', 'password');

// 查积分
const points = await client.getUserPoints();
console.log(points.total_points, points.today_signed);

// 签到
if (!points.today_signed) {
  const result = await client.signin();
  console.log(`获得 ${result.points_earned} 积分`);
}

// 解锁提示词
try {
  const unlock = await client.unlockPrompt(123);
  console.log(`剩余 ${unlock.remaining_points} 积分`);
} catch (err) {
  console.error(err.message);
}

// 消息通知
const unread = await client.getUnreadCount();
const notifications = await client.getNotifications(1, 20);
await client.markAllAsRead();
```

---

## 七、注意事项

1. **Cookie 有效期**：new-api 的 session cookie 有过期时间，过期后需重新登录
2. **并发安全**：签到和解锁接口都有幂等保护，重复调用不会重复扣积分/发奖励
3. **广播消息**：`notifications` 接口会自动合并全员广播消息，无需额外处理
4. **时间戳**：所有时间字段均为 Unix 时间戳（秒），前端需自行转换
5. **积分与额度区分**：`points`（积分）用于签到/解锁，`quota`（额度）用于 API 调用，两者独立
6. **路径规范**：所有接口路径**不带尾部斜杠**，如 `/api/user/points` 而非 `/api/user/points/`

---

## 八、常见问题

**Q: 用户从未签到过，调用 getUserPoints 会报错吗？**
> 不会。首次调用会自动创建积分记录（total_points = 0）。

**Q: 解锁提示词后，getPublicPrompts 会显示解锁状态吗？**
> 不会。需额外调用 `/api/user/unlocked-prompts` 获取已解锁列表，前端自行比对。

**Q: 免费提示词（is_premium=false）调用 unlock 会怎样？**
> 返回错误：`"该提示词免费，无需解锁"`。

**Q: 调用接口返回 401 "New-Api-User header not provided"？**
> 请求头缺少 `New-Api-User`，需传入当前登录用户的 ID。
