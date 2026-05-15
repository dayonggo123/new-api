# SEO 内容接口文档

> 提供给下游系统使用的 SEO 相关接口规范  
> Base URL: `https://heharse.cloud`

---

## 1. 认证方式

### 1.1 公开接口（无需认证）
- `/api/public/*`
- `/prompt/:id`
- `/article/:id`
- `/sitemap.xml`

### 1.2 管理接口（需要管理员权限）
- `/api/prompt/seo/*`
- Header: `Authorization: Bearer <access_token>`
- Header: `New-API-User: <user_id>`

---

## 2. 数据模型

### 2.1 Prompt SEO 字段

```json
{
  "id": 1,
  "title": "提示词标题",
  "content": "中文提示词内容",
  "content_en": "English prompt content",
  "description": "描述",
  "cover_image_url": "https://example.com/image.jpg",
  "author": "@username",
  "model": "Nano banana pro",
  "tags": "[\"摄影\",\"人像\"]",
  "category_id": 3,
  "category_name": "摄影",
  "media_type": "image",
  "status": 1,
  "usage_count": 42,

  // === SEO 核心字段 ===
  "seo_keywords": "AI绘画提示词, 人像摄影prompt, Midjourney教程",
  "intro": "这是一段由AI生成的SEO介绍文案，用于搜索引擎展示...",
  "faq": "[{\"question\":\"如何使用这个提示词？\",\"answer\":\"复制内容到AI工具...\"}]",
  "seo_i18n": "{\"en\":{\"seo_keywords\":\"AI art prompt\",\"intro\":\"...\"}}",

  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `seo_keywords` | string | AI 生成的 SEO 关键词，逗号分隔 |
| `intro` | string | AI 生成的介绍文案（HTML 兼容） |
| `faq` | string(JSON) | FAQ 问答数组，`[{question, answer}]` |
| `seo_i18n` | string(JSON) | 多语言 SEO 内容，`{lang: {seo_keywords, intro, faq}}` |

### 2.2 SEO 审计结果

```json
{
  "overall_score": 85,
  "categories": {
    "title_quality": {
      "score": 90,
      "issues": [],
      "suggestions": ["标题可包含更多关键词"]
    },
    "content_quality": {
      "score": 80,
      "issues": ["内容长度不足"],
      "suggestions": ["补充英文内容"]
    },
    "keyword_usage": { "score": 85, "issues": [], "suggestions": [] },
    "meta_completeness": { "score": 90, "issues": [], "suggestions": [] },
    "technical_seo": { "score": 80, "issues": [], "suggestions": [] }
  },
  "critical_issues": [],
  "quick_wins": ["添加更多标签"]
}
```

---

## 3. 公开 API（无需认证）

### 3.1 获取提示词列表

```
GET /api/public/prompts
```

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |
| `keyword` | string | 否 | 搜索关键词 |
| `category_id` | int | 否 | 分类筛选 |

**返回：**

```json
{
  "success": true,
  "data": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "items": [
      {
        "id": 1,
        "title": "...",
        "seo_keywords": "...",
        "intro": "...",
        // ... 完整 Prompt 字段
      }
    ]
  }
}
```

---

### 3.2 获取单个提示词详情

```
GET /api/public/prompts/:id
```

**返回：**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "title": "...",
    "content": "...",
    "content_en": "...",
    "seo_keywords": "AI绘画, Midjourney, 人像摄影",
    "intro": "这段提示词专为生成高质量人像摄影设计...",
    "faq": "[{\"question\":\"...\",\"answer\":\"...\"}]",
    "seo_i18n": "{\"en\":{\"seo_keywords\":\"...\",\"intro\":\"...\"}}",
    "category_name": "摄影"
  }
}
```

---

### 3.3 获取分类列表

```
GET /api/public/prompt-categories
```

**返回：**

```json
{
  "success": true,
  "data": [
    { "id": 1, "name": "摄影", "icon": "...", "sort_order": 0 }
  ]
}
```

---

## 4. SEO HTML 页面（给搜索引擎爬虫）

### 4.1 提示词 SEO 页面

```
GET /prompt/:id
GET /prompt/:id?lang=en
```

返回完整的 HTML 页面，包含：
- `<title>`、`<meta name="description">`、`<meta name="keywords">`
- Open Graph 标签（`og:title`, `og:description`, `og:image`, `og:url`）
- Twitter Card 标签
- JSON-LD Schema（`Article` + `FAQPage`）
- `<noscript>` 可见内容（给爬虫读取）
- 多语言支持：`?lang=en` 自动切换 SEO 字段

**示例响应头：**
```
Content-Type: text/html; charset=utf-8
Cache-Control: public, max-age=3600
```

---

### 4.2 文章 SEO 页面

```
GET /article/:id
```

结构同上，针对文章类型。

---

### 4.3 Sitemap XML

```
GET /sitemap.xml
```

返回站点地图，包含所有公开提示词和文章的 URL。

---

## 5. SEO 管理 API（需要管理员权限）

### 5.1 获取 SEO 列表

```
GET /api/prompt/seo/list?page=1&page_size=20&keyword=
```

---

### 5.2 获取单个 SEO 详情

```
GET /api/prompt/seo/:id
```

返回完整 Prompt 对象（含所有 SEO 字段）。

---

### 5.3 手动更新 SEO 字段

```
PUT /api/prompt/seo/:id
Content-Type: application/json
```

**请求体：**

```json
{
  "id": 1,
  "seo_keywords": "关键词1, 关键词2",
  "intro": "自定义介绍文案",
  "faq": "[{\"question\":\"Q1\",\"answer\":\"A1\"}]",
  "seo_i18n": "{\"en\":{\"seo_keywords\":\"...\",\"intro\":\"...\"}}"
}
```

---

### 5.4 AI 重新生成 SEO

```
POST /api/prompt/seo/:id/regenerate
```

异步调用 AI 生成 `seo_keywords`、`intro`、`faq`。  
返回：`{ "success": true, "message": "AI 生成任务已启动" }`

---

### 5.5 AI 审计 SEO

```
POST /api/prompt/seo/:id/audit
```

返回 SEO 审计结果：

```json
{
  "success": true,
  "data": {
    "overall_score": 85,
    "categories": { ... },
    "critical_issues": [],
    "quick_wins": ["添加更多标签"]
  }
}
```

---

### 5.6 获取审计历史

```
GET /api/prompt/seo/:id/audits
```

---

### 5.7 获取 SEO 报告（Markdown）

```
GET /api/prompt/seo/:id/report
```

---

### 5.8 SEO 统计概览

```
GET /api/prompt/seo/stats
```

---

### 5.9 SEO 趋势

```
GET /api/prompt/seo/trends?days=30
```

---

### 5.10 低分提示词列表

```
GET /api/prompt/seo/low-score?threshold=60
```

---

### 5.11 批量审计

```
POST /api/prompt/seo/audit-batch
Content-Type: application/json
```

```json
{ "ids": [1, 2, 3] }
```

---

## 6. 自动化流程

### 6.1 创建/更新提示词时自动触发

调用 `POST /api/prompt/` 或 `PUT /api/prompt/:id` 后，系统会**异步**调用 AI 生成 SEO 内容：

```
POST /api/prompt/           -> 创建 -> 后台 AI 生成 SEO
PUT    /api/prompt/:id      -> 更新 -> 后台 AI 重新生成 SEO
```

生成的 SEO 字段会写入 `seo_keywords`、`intro`、`faq`。

### 6.2 多语言 SEO

通过 `seo_i18n` 字段存储多语言版本：

```json
{
  "en": {
    "seo_keywords": "AI art prompt, portrait photography",
    "intro": "English intro...",
    "faq": "[{\"question\":\"...\",\"answer\":\"...\"}]"
  },
  "ja": {
    "seo_keywords": "...",
    "intro": "...",
    "faq": "..."
  }
}
```

访问 `/prompt/:id?lang=en` 时，系统会自动应用对应语言的 SEO 字段。

---

## 7. 下游使用建议

### 7.1 采集入库后自动触发 SEO 生成

扩展采集提示词入库后，可调用：

```
POST /api/prompt/seo/:id/regenerate
```

触发 AI 自动生成 SEO 内容。

### 7.2 批量优化低分内容

```
GET /api/prompt/seo/low-score?threshold=60
// 获取低分列表后，对每个调用 regenerate
```

### 7.3 集成 Sitemap

下游系统可定期抓取 `https://heharse.cloud/sitemap.xml` 获取全部可索引 URL。
