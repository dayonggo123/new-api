# 多语言接口测试指南

## 一、环境准备

确保已部署最新代码，然后：

```bash
# 1. 确认服务启动
curl https://your-domain.com/api/status

# 2. 准备测试账号（管理员）
# 登录后获取 Cookie 中的 session
```

---

## 二、预设提示词接口测试

### 测试 1：匿名用户默认中文

```bash
curl -s "https://your-domain.com/api/public/preset-prompts" | jq '.data[0]'
```

**预期结果：**
```json
{
  "id": 2,
  "name": "图片内容详细描述",
  "system_prompt": "你是一位专业的视觉分析专家...",
  "category": "图片分析",
  "i18n": ""
}
```
> `i18n` 字段被隐藏，内容保持默认中文

---

### 测试 2：匿名用户显式指定英文

```bash
curl -s "https://your-domain.com/api/public/preset-prompts?lang=en" | jq '.data[0]'
```

**预期结果：**
```json
{
  "id": 2,
  "name": "Detailed Description of Image Content",
  "system_prompt": "You are a professional visual analysis expert...",
  "category": "Image Analysis",
  "i18n": ""
}
```
> `name`、`system_prompt`、`user_prompt`、`description`、`category` 都已替换为英文

---

### 测试 3：匿名用户指定德语

```bash
curl -s "https://your-domain.com/api/public/preset-prompts?lang=de" | jq '.data[0]'
```

**预期结果：**
```json
{
  "id": 2,
  "name": "Detaillierte Beschreibung des Bildinhalts",
  "category": "Bildanalyse",
  "i18n": ""
}
```

---

### 测试 4：已登录用户自动语言偏好（英文）

```bash
# 先设置用户语言为英文
curl -X PUT "https://your-domain.com/api/user/self" \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{"language": "en"}'

# 再调用公开接口（不传 lang）
curl -s "https://your-domain.com/api/public/preset-prompts" \
  -H "Cookie: session=YOUR_SESSION" | jq '.data[0].name'
```

**预期结果：** `"Detailed Description of Image Content"`

---

### 测试 5：已登录用户强制覆盖语言

```bash
# 用户语言偏好是 en，但强制传 lang=fr
curl -s "https://your-domain.com/api/public/preset-prompts?lang=fr" \
  -H "Cookie: session=YOUR_SESSION" | jq '.data[0].name'
```

**预期结果：** `"Description détaillée du contenu de l'image"`（法语）

> `?lang=` 参数优先级高于用户语言偏好

---

### 测试 6：缺失翻译的 fallback

```bash
# 调用韩文，但某条预设提示词没有韩文翻译
curl -s "https://your-domain.com/api/public/preset-prompts?lang=ko" | jq '.data[0].name'
```

**预期结果：** `"图片内容详细描述"`（默认中文，不报错）

---

## 三、消息通知接口测试

### 测试 7：用户消息自动语言

```bash
# 1. 管理员发送一条带多语言翻译的公告
# 在管理后台 → 消息管理 → 发布消息
# 中文标题：系统维护通知
# 英文标题：System Maintenance Notice
# 发送给全员

# 2. 用户 A（语言偏好 zh-CN）查看消息
curl -s "https://your-domain.com/api/notifications" \
  -H "Cookie: session=USER_A_SESSION" | jq '.data.items[0].title'
```

**预期结果：** `"系统维护通知"`

```bash
# 3. 用户 B（语言偏好 en）查看同一条消息
curl -s "https://your-domain.com/api/notifications" \
  -H "Cookie: session=USER_B_SESSION" | jq '.data.items[0].title'
```

**预期结果：** `"System Maintenance Notice"`

---

### 测试 8：消息内容翻译

```bash
curl -s "https://your-domain.com/api/notifications" \
  -H "Cookie: session=USER_B_SESSION" | jq '.data.items[0] | {title, content}'
```

**预期结果：**
```json
{
  "title": "System Maintenance Notice",
  "content": "We will perform system maintenance at 2:00 AM UTC..."
}
```

---

## 四、管理端多语言编辑测试

### 测试 9：预设提示词多语言编辑

1. 打开管理后台 → 预设提示词管理
2. 点击"编辑"某条提示词
3. 切换到 **English** Tab
4. 看到已翻译的字段（如果之前翻译过）
5. 点击 🌐"重新翻译"按钮
6. 等待翻译完成，字段自动填充
7. 点击"确定"保存
8. 再次调用接口验证：

```bash
curl -s "https://your-domain.com/api/public/preset-prompts?lang=en" | jq '.data[] | select(.id==2) | .name'
```

**预期结果：** 返回重新翻译后的英文名称

---

### 测试 10：消息管理多语言编辑

1. 打开管理后台 → 消息管理
2. 点击某条消息的"编辑"
3. 切换到 **English** Tab
4. 修改英文标题
5. 点击"更新"
6. 英文用户查看消息，应看到更新后的内容

```bash
curl -s "https://your-domain.com/api/notifications" \
  -H "Cookie: session=EN_USER_SESSION" | jq '.data.items[0].title'
```

---

## 五、边界情况测试

### 测试 11：空 i18n 数据

```bash
# 某条预设提示词完全没有翻译数据
curl -s "https://your-domain.com/api/public/preset-prompts?lang=es" | jq '.data[] | select(.i18n=="") | .name'
```

**预期结果：** 返回默认中文名称，不报错

---

### 测试 12： category 翻译

```bash
# 验证 category 字段也被翻译了
curl -s "https://your-domain.com/api/public/preset-prompts?lang=en" | jq '.data[0].category'
```

**预期结果：** `"Image Analysis"`（如果该提示词的英文 category 已翻译）

---

### 测试 13： admin 列表接口隐藏 i18n

```bash
# 管理员列表接口也隐藏了 i18n 原始 JSON
curl -s "https://your-domain.com/api/admin/notifications" \
  -H "Cookie: session=ADMIN_SESSION" | jq '.data.items[0].i18n'
```

**预期结果：** `""`（空字符串）

---

## 六、快速验证脚本

```bash
#!/bin/bash
BASE="https://your-domain.com"

echo "=== Test 1: Default Chinese ==="
curl -s "$BASE/api/public/preset-prompts" | jq -r '.data[0].name'

echo "=== Test 2: English ==="
curl -s "$BASE/api/public/preset-prompts?lang=en" | jq -r '.data[0].name'

echo "=== Test 3: German ==="
curl -s "$BASE/api/public/preset-prompts?lang=de" | jq -r '.data[0].name'

echo "=== Test 4: Category translation ==="
curl -s "$BASE/api/public/preset-prompts?lang=en" | jq -r '.data[0].category'

echo "=== Test 5: Missing translation fallback ==="
curl -s "$BASE/api/public/preset-prompts?lang=ko" | jq -r '.data[0].name'

echo "=== Test 6: i18n field hidden ==="
curl -s "$BASE/api/public/preset-prompts?lang=en" | jq -r '.data[0].i18n'
```

---

## 七、常见问题排查

| 现象 | 原因 | 解决 |
|---|---|---|
| 返回还是中文 | `lang` 参数未传 + 用户未登录 | 传 `?lang=en` 或登录 |
| category 还是中文 | 该提示词的 category 没有翻译 | 在管理端编辑并翻译 category |
| `name is required` 错误 | 非默认 Tab 提交时 Form 验证失败 | 已修复，重新部署 |
| `i18n` 字段仍返回 | 接口未隐藏 | 确认已部署最新代码 |
| 翻译按钮无反应 | `translating` 未定义 | 已修复，重新部署 |
