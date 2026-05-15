# New-API 对接文档 — 提示词 / 文章 / 模型接口

> 本文档面向第三方系统对接，涵盖 **提示词库(Prompt)**、**文章(Article)**、**模型(Model)** 三大模块的管理端与公共端接口。

---

## 1. 通用约定

### Base URL
```
https://your-domain.com/api
```

### 认证方式
- **管理端接口**：需在请求头携带管理员 JWT Token
  ```
  Authorization: Bearer <admin_jwt_token>
  ```
- **公共端接口**：无需认证

### 统一响应格式
```json
{
  "success": true,
  "message": "",
  "data": { }
}
```

### 分页参数（Query）
| 参数 | 类型 | 说明 |
|------|------|------|
| `p` | int | 页码，从 1 开始 |
| `page_size` | int | 每页条数，默认 10 |

分页响应 data 结构：
```json
{
  "items": [],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

---

## 2. 提示词库（Prompt）接口

### 2.1 管理端 — 提示词分类

#### 获取分类列表（分页）
```
GET /api/prompt-category/?p=1&page_size=20
```

#### 获取全部分类（不分页，仅启用的）
```
GET /api/prompt-category/all
```

#### 获取单个分类
```
GET /api/prompt-category/:id
```

#### 创建分类
```
POST /api/prompt-category/
```
**请求体：**
```json
{
  "name": "图像生成",
  "description": "用于图像生成的提示词",
  "icon": "icon-url",
  "sort_order": 0,
  "status": 1
}
```

#### 更新分类
```
PUT /api/prompt-category/
```
**请求体：**
```json
{
  "id": 1,
  "name": "图像生成",
  "description": "...",
  "icon": "...",
  "sort_order": 1,
  "status": 1
}
```

#### 删除分类
```
DELETE /api/prompt-category/:id
```

---

### 2.2 管理端 — 提示词

#### 获取提示词列表（分页+搜索）
```
GET /api/prompt?p=1&page_size=20&keyword=xxx&category_id=1
```

#### 获取单个提示词
```
GET /api/prompt/:id
```

#### 创建提示词
```
POST /api/prompt/
```
**请求体：**
```json
{
  "category_id": 1,
  "title": "卡通头像生成器",
  "content": "请生成一个卡通风格的头像，描述如下：...",
  "content_en": "English version...",
  "description": "快速生成卡通头像",
  "cover_image_url": "https://...",
  "author": "@dayong",
  "model": "DALL-E",
  "variables": "[{\"name\":\"style\",\"type\":\"select\",\"options\":[\"卡通\",\"写实\"]}]",
  "tags": "[\"头像\",\"AI绘画\"]",
  "sort_order": 0,
  "status": 1,
  "media_type": "image",
  "is_premium": false,
  "unlock_cost": 0,
  "i18n": "{}",
  "seo_i18n": "{}"
}
```

#### 更新提示词
```
PUT /api/prompt/
```
**请求体：** 同创建，需包含 `id`

#### 删除提示词
```
DELETE /api/prompt/:id
```

---

### 2.3 管理端 — 提示词 SEO 管理

#### 获取 SEO 列表（分页+搜索）
```
GET /api/prompt/seo/list?p=1&page_size=20&keyword=xxx
```
**响应字段：**
```json
{
  "id": 1,
  "title": "...",
  "category_name": "...",
  "seo_keywords": "...",
  "intro": "...",
  "faq": "[{\"question\":\"...\",\"answer\":\"...\"}]",
  "seo_i18n": "{}",
  "audit_score": 85,
  "status": 1,
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

#### 获取单个提示词 SEO 详情
```
GET /api/prompt/seo/:id
```

#### 更新 SEO 字段
```
PUT /api/prompt/seo/:id
```
**请求体：**
```json
{
  "id": 1,
  "seo_keywords": "关键词1, 关键词2",
  "intro": "介绍文案...",
  "faq": "[{\"question\":\"Q1\",\"answer\":\"A1\"}]",
  "seo_i18n": "{\"en\":{\"seo_keywords\":\"...\",\"intro\":\"...\",\"faq\":\"...\"}}"
}
```

#### AI 重新生成 SEO
```
POST /api/prompt/seo/:id/regenerate
```

#### AI 审计 SEO
```
POST /api/prompt/seo/:id/audit
```
**响应：**
```json
{
  "overall_score": 85,
  "categories": {
    "completeness": { "score": 90, "issues": [], "suggestions": [] },
    "keyword_quality": { "score": 80, "issues": [], "suggestions": [] },
    "intro_quality": { "score": 85, "issues": [], "suggestions": [] },
    "faq_quality": { "score": 88, "issues": [], "suggestions": [] },
    "structured_data": { "score": 82, "issues": [], "suggestions": [] }
  },
  "critical_issues": ["..."],
  "quick_wins": ["..."]
}
```

#### 获取审计历史
```
GET /api/prompt/seo/:id/audits?limit=10
```

#### 获取审计报告（最新一条）
```
GET /api/prompt/seo/:id/report
```

#### 获取统计概览
```
GET /api/prompt/seo/stats
```
**响应：**
```json
{
  "seo_coverage": 75,
  "with_seo": 75,
  "total_prompts": 100,
  "audit_coverage": 60,
  "with_audit": 60,
  "average_score": 78.5,
  "score_distribution": [
    { "range": "excellent", "count": 20 },
    { "range": "good", "count": 30 },
    { "range": "average", "count": 25 },
    { "range": "poor", "count": 5 }
  ]
}
```

---

### 2.4 公共端 — 提示词（无需认证）

#### 获取公共提示词列表
```
GET /api/public/prompts?p=1&page_size=20&keyword=xxx&category_id=1
```

#### 获取单个公共提示词
```
GET /api/public/prompts/:id
```

#### 获取公共分类列表
```
GET /api/public/prompt-categories
```

---

## 3. 文章（Article）接口

### 3.1 管理端 — 文章分类

#### 获取分类列表（分页）
```
GET /api/admin/article-categories?p=1&page_size=20
```

#### 获取单个分类
```
GET /api/admin/article-categories/:id
```

#### 创建分类
```
POST /api/admin/article-categories
```
**请求体：**
```json
{
  "name": "技术博客",
  "description": "技术类文章",
  "icon": "icon-url",
  "sort_order": 0,
  "status": 1
}
```

#### 更新分类
```
PUT /api/admin/article-categories/:id
```
**请求体：** 同创建，需包含 `id`

#### 删除分类
```
DELETE /api/admin/article-categories/:id
```

---

### 3.2 管理端 — 文章

#### 获取文章列表（分页+搜索+筛选）
```
GET /api/admin/articles?p=1&page_size=20&keyword=xxx&category_id=1&status=1
```

#### 获取单个文章
```
GET /api/admin/articles/:id
```

#### 创建文章
```
POST /api/admin/articles
```
**请求体：**
```json
{
  "category_id": 1,
  "title": "文章标题",
  "slug": "article-slug",
  "content": "# 正文内容...",
  "summary": "文章摘要",
  "cover_image_url": "https://...",
  "author": "作者名",
  "tags": "[\"标签1\",\"标签2\"]",
  "status": 1,
  "is_featured": false,
  "seo_title": "SEO标题",
  "seo_description": "SEO描述",
  "seo_keywords": "关键词1, 关键词2",
  "i18n": "{}",
  "seo_i18n": "{}"
}
```

#### 更新文章
```
PUT /api/admin/articles/:id
```
**请求体：** 同创建，需包含 `id`

#### 删除文章
```
DELETE /api/admin/articles/:id
```

#### AI 写文章
```
POST /api/admin/articles/generate
```
**请求体：**
```json
{
  "title": "可选标题",
  "prompt": "写一篇关于AI的文章",
  "reference_url": "https://参考链接.com",
  "language": "zh"
}
```
**响应：**
```json
{
  "title": "生成的标题",
  "content": "生成的正文",
  "summary": "摘要",
  "tags": "[\"标签\"]",
  "cover_image_url": "...",
  "author": "AI",
  "seo_title": "...",
  "seo_description": "...",
  "seo_keywords": "..."
}
```

#### AI 生成文章配图
```
POST /api/admin/articles/generate-images
```
**请求体：**
```json
{
  "prompt": "生成提示词",
  "n": 2,
  "size": "1024x1024"
}
```

---

### 3.3 管理端 — 文章 SEO 管理

#### 获取文章 SEO 列表（分页+搜索）
```
GET /api/article/seo/list?p=1&page_size=20&keyword=xxx
```
**响应字段：**
```json
{
  "id": 1,
  "title": "...",
  "category_name": "...",
  "seo_title": "...",
  "seo_description": "...",
  "seo_keywords": "...",
  "seo_i18n": "{}",
  "audit_score": 85,
  "status": 1,
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

#### 获取单个文章 SEO 详情
```
GET /api/article/seo/:id
```

#### 更新 SEO 字段
```
PUT /api/article/seo/:id
```
**请求体：**
```json
{
  "id": 1,
  "seo_title": "SEO标题",
  "seo_description": "SEO描述",
  "seo_keywords": "关键词1, 关键词2",
  "seo_i18n": "{\"en\":{\"seo_title\":\"...\",\"seo_description\":\"...\",\"seo_keywords\":\"...\"}}"
}
```

#### AI 重新生成 SEO
```
POST /api/article/seo/:id/regenerate
```

#### AI 审计 SEO
```
POST /api/article/seo/:id/audit
```
**响应：**
```json
{
  "overall_score": 85,
  "categories": {
    "completeness": { "score": 90, "issues": [], "suggestions": [] },
    "keyword_quality": { "score": 80, "issues": [], "suggestions": [] },
    "title_quality": { "score": 85, "issues": [], "suggestions": [] },
    "description_quality": { "score": 88, "issues": [], "suggestions": [] },
    "technical": { "score": 82, "issues": [], "suggestions": [] }
  },
  "critical_issues": ["..."],
  "quick_wins": ["..."]
}
```

#### 获取审计历史
```
GET /api/article/seo/:id/audits?limit=10
```

#### 获取审计报告（最新一条）
```
GET /api/article/seo/:id/report
```

#### 获取统计概览
```
GET /api/article/seo/stats
```
**响应：**
```json
{
  "seo_coverage": 75,
  "with_seo": 75,
  "total_articles": 100,
  "audit_coverage": 60,
  "with_audit": 60,
  "average_score": 78.5,
  "score_distribution": [
    { "range": "excellent", "count": 20 },
    { "range": "good", "count": 30 },
    { "range": "average", "count": 25 },
    { "range": "poor", "count": 5 }
  ]
}
```

#### 获取低分文章
```
GET /api/article/seo/low-score?threshold=60&limit=20
```

---

### 3.4 公共端 — 文章（无需认证）

#### 获取公共文章列表
```
GET /api/public/articles?p=1&page_size=20&keyword=xxx&category_id=1
```

#### 获取单个公共文章（ID）
```
GET /api/public/articles/:id
```

#### 获取单个公共文章（Slug）
```
GET /api/public/articles/slug/:slug
```

#### 获取公共分类列表
```
GET /api/public/article-categories
```

---

## 4. 模型（Model）接口

### 4.1 管理端 — 模型元数据

#### 获取模型列表（分页）
```
GET /api/models?p=1&page_size=20
```
**响应字段：**
```json
{
  "id": 1,
  "model_name": "gpt-4o",
  "description": "OpenAI GPT-4o",
  "icon": "...",
  "tags": "chat,vision",
  "vendor_id": 1,
  "endpoints": "[{\"type\":\"chat\",\"url\":\"...\"}]",
  "status": 1,
  "sync_official": 1,
  "bound_channels": [{"id":1,"name":"OpenAI"}],
  "enable_groups": ["default","vip"],
  "quota_types": [1,2],
  "matched_models": ["gpt-4o-2024-05-13"],
  "matched_count": 1,
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

#### 搜索模型
```
GET /api/models/search?keyword=gpt&vendor=1&p=1&page_size=20
```

#### 获取单个模型
```
GET /api/models/:id
```

#### 创建模型
```
POST /api/models/
```
**请求体：**
```json
{
  "model_name": "gpt-4o",
  "description": "OpenAI GPT-4o",
  "icon": "https://...",
  "tags": "chat,vision",
  "vendor_id": 1,
  "endpoints": "[{\"type\":\"chat\",\"url\":\"https://api.openai.com/v1/chat/completions\"}]",
  "status": 1,
  "sync_official": 1
}
```

#### 更新模型
```
PUT /api/models/
```
**请求体：** 同创建，需包含 `id`

> 支持 `?status_only=true` 参数，仅更新状态字段。

#### 删除模型
```
DELETE /api/models/:id
```

#### 获取缺失模型（上游有但本地未录入）
```
GET /api/models/missing
```

#### 同步上游模型预览
```
GET /api/models/sync_upstream/preview
```

#### 同步上游模型
```
POST /api/models/sync_upstream
```

---

## 5. 核心数据结构

### 5.1 Prompt（提示词）
```json
{
  "id": 1,
  "category_id": 1,
  "title": "标题",
  "content": "提示词内容",
  "content_en": "英文内容",
  "description": "描述",
  "cover_image_url": "...",
  "author": "@作者",
  "model": "ChatGPT",
  "variables": "JSON数组",
  "tags": "JSON数组",
  "sort_order": 0,
  "status": 1,
  "media_type": "image",
  "is_premium": false,
  "unlock_cost": 0,
  "seo_keywords": "...",
  "intro": "...",
  "faq": "JSON数组",
  "i18n": "{}",
  "seo_i18n": "{}",
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

### 5.2 Article（文章）
```json
{
  "id": 1,
  "category_id": 1,
  "title": "文章标题",
  "slug": "article-slug",
  "content": "# Markdown正文",
  "summary": "摘要",
  "cover_image_url": "...",
  "author": "作者",
  "tags": "JSON数组",
  "status": 1,
  "is_featured": false,
  "view_count": 100,
  "seo_title": "...",
  "seo_description": "...",
  "seo_keywords": "...",
  "i18n": "{}",
  "seo_i18n": "{}",
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

### 5.3 Model（模型）
```json
{
  "id": 1,
  "model_name": "gpt-4o",
  "description": "...",
  "icon": "...",
  "tags": "chat,vision",
  "vendor_id": 1,
  "endpoints": "JSON数组",
  "status": 1,
  "sync_official": 1,
  "bound_channels": [],
  "enable_groups": [],
  "quota_types": [],
  "matched_models": [],
  "matched_count": 0,
  "created_time": 1700000000,
  "updated_time": 1700000000
}
```

---

## 6. 状态码说明

| 状态码 | 说明 |
|--------|------|
| `status=1` | 启用 |
| `status=2` | 禁用 |
| `status=0` | 不限制（查询参数） |

---

*文档生成时间：2026-05-14*
