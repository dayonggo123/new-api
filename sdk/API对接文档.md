# new-api 积分/签到/消息系统 — 下游对接文档

> 提供给 ewapi 等下游应用的前端/后端对接说明

---

## 一、基本信息

| 项目 | 说明 |
|------|------|
| **接口基地址** | `https://your-domain.com/api` |
| **认证方式** | `Bearer Token`（从登录接口获取） |
| **请求格式** | `application/json` |
| **响应格式** | 统一 JSON：`{ success: boolean, message: string, data: any }` |

---

## 二、认证方式

### 1. 登录获取 Token

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
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": { "id": 1, "username": "xxx", ... }
  }
}
```

### 2. 后续请求带 Token

```http
GET /api/user/points
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

---

## 三、接口清单

### 3.1 用户积分 & 签到

#### 获取积分和签到状态
```http
GET /api/user/points
Authorization: Bearer <token>
```

**响应**：
```json
{
  "success": true,
  "data": {
    "total_points": 150,
    "consecutive_days": 3,
    "last_signin_date": 1750000000,
    "today_signed": false,
    "next_signin_points": 12
  }
}
```

---

#### 执行签到
```http
POST /api/user/signin
Authorization: Bearer <token>
```

**响应**：
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

**错误响应**：
```json
{
  "success": false,
  "message": "今日已签到"
}
```

---

#### 签到历史
```http
GET /api/user/signin-history?page=1&page_size=20
Authorization: Bearer <token>
```

**响应**：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "signin_date": 1750000000,
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

#### 解锁提示词
```http
POST /api/user/unlock-prompt
Authorization: Bearer <token>
Content-Type: application/json

{
  "prompt_id": 123
}
```

**响应**：
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

**错误响应**：
```json
{ "success": false, "message": "积分不足，需要 50 积分" }
{ "success": false, "message": "已解锁该提示词" }
{ "success": false, "message": "该提示词免费，无需解锁" }
```

---

#### 已解锁提示词列表
```http
GET /api/user/unlocked-prompts?page=1&page_size=20
Authorization: Bearer <token>
```

**响应**：
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
        "unlocked_at": 1750000000
      }
    ],
    "total": 10
  }
}
```

---

### 3.3 消息通知

#### 获取消息列表
```http
GET /api/notifications?page=1&page_size=20
Authorization: Bearer <token>
```

**响应**：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "title": "系统维护通知",
        "content": "系统将于今晚进行维护...",
        "type": "system",
        "is_read": false,
        "action_url": "",
        "created_time": 1750000000
      }
    ],
    "total": 100
  }
}
```

> **注意**：列表会自动包含全员广播消息（`user_id = 0`）和该用户的专属消息。

---

#### 获取未读消息数
```http
GET /api/notifications/unread-count
Authorization: Bearer <token>
```

**响应**：
```json
{
  "success": true,
  "data": { "unread_count": 5 }
}
```

---

#### 标记单条已读
```http
POST /api/notifications/1/read
Authorization: Bearer <token>
```

---

#### 标记全部已读
```http
POST /api/notifications/read-all
Authorization: Bearer <token>
```

**响应**：
```json
{
  "success": true,
  "data": { "marked_count": 5 }
}
```

---

### 3.4 公开提示词（新增字段）

原有的公开提示词接口已自动扩展 `is_premium` 和 `unlock_cost` 字段：

```http
GET /api/public/prompts?page=1&page_size=20&category_id=0&keyword=xxx
```

**响应**（单条提示词示例）：
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": 1,
        "title": "提示词标题",
        "content": "提示词内容...",
        "cover_image_url": "https://...",
        "category_name": "分类名",
        "is_premium": true,
        "unlock_cost": 50
      }
    ],
    "total": 100
  }
}
```

---

## 四、前端集成建议

### 4.1 推荐流程

```
1. 用户访问 ewapi → 检测本地是否有 token
2. 无 token → 跳转到登录页 → 调用 /api/user/login → 保存 token
3. 有 token → 调用 /api/user/points 显示积分和签到按钮
4. 浏览提示词列表 → is_premium=true 的显示"解锁"按钮
5. 点击解锁 → 调用 /api/user/unlock-prompt → 刷新积分和解锁状态
```

### 4.2 签到按钮状态

| 状态 | 表现 |
|------|------|
| `today_signed = false` | 显示"签到"按钮，可点击 |
| `today_signed = true` | 显示"已签到"，禁用按钮，显示"连续X天" |

### 4.3 提示词卡片状态

```
if (prompt.is_premium) {
  if (已解锁) {
    显示"已解锁"，可正常使用
  } else {
    显示"解锁（需X积分）"，点击调用解锁接口
  }
} else {
  免费提示词，直接可用
}
```

---

## 五、SDK 使用（TypeScript）

已提供封装好的 SDK：`sdk/ewapi-client.ts`

```typescript
import { EwapiClient } from './sdk/ewapi-client';

const client = new EwapiClient('https://your-domain.com');

// 登录
const { token } = await client.login('username', 'password');
client.setToken(token);

// 积分签到
const points = await client.getUserPoints();
if (!points.today_signed) {
  await client.signin();
}

// 解锁提示词
try {
  await client.unlockPrompt(123);
} catch (err) {
  alert(err.message);
}
```

---

## 六、注意事项

1. **Token 有效期**：new-api 的 JWT Token 有过期时间，过期后需重新登录
2. **并发安全**：签到和解锁接口都有幂等保护，重复调用不会重复扣积分/发奖励
3. **广播消息**：`notifications` 接口会自动合并全员广播消息，无需额外处理
4. **时间戳**：所有时间字段均为 Unix 时间戳（秒），前端需自行转换
5. **积分与额度区分**：`points`（积分）用于签到/解锁，`quota`（额度）用于 API 调用，两者独立

---

## 七、常见问题

**Q: 用户从未签到过，调用 getUserPoints 会报错吗？**
> 不会。首次调用会自动创建积分记录（total_points = 0）。

**Q: 解锁提示词后，getPublicPrompts 会显示解锁状态吗？**
> 不会。需额外调用 `/api/user/unlocked-prompts` 获取已解锁列表，前端自行比对。

**Q: 免费提示词（is_premium=false）调用 unlock 会怎样？**
> 返回错误：`"该提示词免费，无需解锁"`。
