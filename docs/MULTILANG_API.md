# 多语言内容接口对接文档

> 适用于：外部客户端、SDK、Tauri/Electron 桌面端、移动端 App 等下游应用

---

## 一、预设提示词接口

### 接口地址

```
GET /api/public/preset-prompts
```

### 认证

- **公开访问**：无需登录，直接调用
- **已登录用户**：传 Cookie Session 或 `Authorization` Header，可不传 `lang` 参数，系统自动根据用户语言偏好返回

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `lang` | string | 否 | 语言代码，如 `en`、`fr`、`ja`。不传时，已登录用户自动读取账户语言偏好；未登录则返回默认中文 |

### 支持的语言代码

| 语言 | 代码 |
|---|---|
| 中文（默认） | `zh` / `zh-CN` / `zh-TW` |
| English | `en` |
| Français | `fr` |
| Русский | `ru` |
| 日本語 | `ja` |
| Tiếng Việt | `vi` |
| 한국어 | `ko` |
| Español | `es` |
| Deutsch | `de` |
| Italiano | `it` |
| Português | `pt` |
| العربية | `ar` |

### 返回字段

系统会根据 `lang` 自动替换以下字段为对应语言内容（若该语言无翻译，则保持默认中文不变）：

| 字段 | 说明 |
|---|---|
| `id` | 提示词 ID |
| `name` | 提示词名称（已翻译） |
| `system_prompt` | 系统提示词（已翻译） |
| `user_prompt` | 用户提示词（已翻译） |
| `description` | 描述（已翻译） |
| `category` | 分类（已翻译） |
| `status` | 状态：1=启用，2=禁用 |
| `sort_order` | 排序权重 |

> ⚠️ `i18n` 原始 JSON 字段不会返回，已内联替换到各字段中

### 调用示例

#### cURL

```bash
# 1. 匿名用户显式指定英文
curl "https://your-domain.com/api/public/preset-prompts?lang=en"

# 2. 已登录用户（自动根据账户语言偏好）
curl "https://your-domain.com/api/public/preset-prompts" \
  -H "Cookie: session=your_session_cookie"

# 3. 已登录用户强制覆盖语言
curl "https://your-domain.com/api/public/preset-prompts?lang=fr" \
  -H "Cookie: session=your_session_cookie"
```

#### JavaScript / TypeScript

```typescript
async function getPresetPrompts(lang?: string): Promise<PresetPrompt[]> {
  const params = lang ? `?lang=${lang}` : '';
  const res = await fetch(`/api/public/preset-prompts${params}`, {
    credentials: 'include', // 自动携带 Cookie
  });
  const data = await res.json();
  return data.data; // PresetPrompt[]
}

// 使用示例
const prompts = await getPresetPrompts('en');
console.log(prompts[0].name);        // "Detailed Description of Image Content"
console.log(prompts[0].category);    // "Image Analysis"
```

#### Python

```python
import requests

# 显式指定语言
r = requests.get("https://your-domain.com/api/public/preset-prompts?lang=en")
prompts = r.json()["data"]
print(prompts[0]["name"])       # "Detailed Description of Image Content"
print(prompts[0]["category"])   # "Image Analysis"
```

---

## 二、消息通知接口

### 接口地址

```
GET /api/notifications
```

### 认证

**必须登录**，系统根据当前登录用户的 `users.setting.language` 自动返回对应语言内容。

无需传 `lang` 参数。

### 返回字段（已翻译）

| 字段 | 说明 |
|---|---|
| `id` | 消息 ID |
| `title` | 标题（已根据用户语言偏好翻译） |
| `content` | 内容（已根据用户语言偏好翻译） |
| `type` | 类型：system / promotion / announcement / task_status |
| `is_read` | 是否已读 |
| `action_url` | 跳转链接 |
| `created_time` | 创建时间戳 |

### 调用示例

#### cURL

```bash
curl "https://your-domain.com/api/notifications?page=1&page_size=10" \
  -H "Cookie: session=your_session_cookie"
```

#### JavaScript / TypeScript

```typescript
async function getNotifications() {
  const res = await fetch('/api/notifications?page=1&page_size=10', {
    credentials: 'include',
  });
  const data = await res.json();
  return data.data.items; // Notification[]
}
```

---

## 三、用户语言偏好设置

下游如需让用户自主选择语言，可调用以下接口更新用户语言偏好：

```
PUT /api/user/self
Content-Type: application/json

{"language": "en"}
```

支持的语言代码：`zh-CN`, `zh-TW`, `en`, `fr`, `ru`, `ja`, `vi`

设置后，后续调用 `/api/notifications` 和不传 `lang` 的 `/api/public/preset-prompts` 将自动返回对应语言内容。

---

## 四、FAQ

**Q1：如果某个语言没有翻译，返回什么？**
> 返回默认中文内容，不会报错或返回空。

**Q2：预设提示词接口必须传 `lang` 吗？**
> 不是必须的。未传 `lang` 时：
> - 已登录用户 → 自动使用账户语言偏好
> - 未登录用户 → 返回默认中文

**Q3：消息接口可以传 `lang` 强制覆盖吗？**
> 目前不支持，消息接口严格根据用户账户语言偏好返回。如需覆盖，建议先调用 `PUT /api/user/self` 修改语言偏好。

**Q4：前端如何知道当前用户语言？**
> 调用 `GET /api/user/self` 读取 `setting.language` 字段。
