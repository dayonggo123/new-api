# New-API 下游读取对接文档

> 本文档面向**下游读取方**，只包含获取提示词、文章、模型及价格的**读取接口**。

---

## 通用约定

### Base URL
```
https://your-domain.com/api
```

### 认证方式
- **公共接口**：无需认证
- **Pricing / Ratio 接口**：可选认证（不携带 Token 也可访问公共数据）

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

---

## 1. 提示词库（Prompt）— 公共接口（无需认证）

### 获取公共提示词列表
```
GET /api/public/prompts?p=1&page_size=20&keyword=xxx&category_id=1
```

**响应 data 字段：**
```json
{
  "items": [
    {
      "id": 1,
      "category_id": 1,
      "title": "卡通头像生成器",
      "content": "请生成一个卡通风格的头像...",
      "content_en": "English version...",
      "description": "快速生成卡通头像",
      "cover_image_url": "https://...",
      "author": "@dayong",
      "model": "DALL-E",
      "variables": "[{\"name\":\"style\",\"type\":\"select\"}]",
      "tags": "[\"头像\",\"AI绘画\"]",
      "sort_order": 0,
      "status": 1,
      "media_type": "image",
      "seo_keywords": "...",
      "intro": "...",
      "faq": "[{\"question\":\"Q1\",\"answer\":\"A1\"}]",
      "i18n": "{}",
      "seo_i18n": "{}",
      "created_time": 1700000000,
      "updated_time": 1700000000
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

### 获取单个公共提示词
```
GET /api/public/prompts/:id
```

### 获取公共分类列表
```
GET /api/public/prompt-categories
```

**响应：**
```json
[
  {
    "id": 1,
    "name": "图像生成",
    "description": "...",
    "icon": "...",
    "sort_order": 0,
    "status": 1
  }
]
```

---

## 2. 文章（Article）— 公共接口（无需认证）

### 获取公共文章列表
```
GET /api/public/articles?p=1&page_size=20&keyword=xxx&category_id=1
```

**响应 data 字段：**
```json
{
  "items": [
    {
      "id": 1,
      "category_id": 1,
      "title": "文章标题",
      "slug": "article-slug",
      "content": "# Markdown正文",
      "summary": "文章摘要",
      "cover_image_url": "https://...",
      "author": "作者名",
      "tags": "[\"标签1\",\"标签2\"]",
      "status": 1,
      "is_featured": false,
      "view_count": 100,
      "seo_title": "...",
      "seo_description": "...",
      "seo_keywords": "...",
      "i18n": "{}",
      "seo_i18n": "{}",
      "category_name": "分类名称",
      "created_time": 1700000000,
      "updated_time": 1700000000
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

### 获取单个公共文章（ID）
```
GET /api/public/articles/:id
```

### 获取单个公共文章（Slug）
```
GET /api/public/articles/slug/:slug
```

### 获取公共分类列表
```
GET /api/public/article-categories
```

---

## 3. 模型及价格 — 读取接口

### 获取模型定价表（核心接口）
```
GET /api/pricing
```

> 认证可选。不携带 Token 返回全部公开定价；携带 Token 则按用户组过滤可见模型。

**响应 data 字段：**
```json
[
  {
    "model_name": "gpt-4o",
    "description": "OpenAI GPT-4o",
    "icon": "https://...",
    "tags": "chat,vision",
    "vendor_id": 1,
    "quota_type": 1,
    "model_ratio": 15.0,
    "model_price": 0.0,
    "owner_by": "OpenAI",
    "completion_ratio": 2.0,
    "cache_ratio": 0.5,
    "create_cache_ratio": 1.0,
    "image_ratio": null,
    "audio_ratio": null,
    "audio_completion_ratio": null,
    "enable_groups": ["default", "vip"],
    "supported_endpoint_types": ["chat", "image"],
    "pricing_version": "v1"
  }
]
```

**字段说明：**
| 字段 | 说明 |
|------|------|
| `model_name` | 模型名称 |
| `model_ratio` | 输入倍率 |
| `model_price` | 固定价格（当 model_price > 0 时按固定价计费） |
| `completion_ratio` | 输出倍率（输入倍率 × 此值 = 输出计费） |
| `cache_ratio` | 缓存命中倍率 |
| `create_cache_ratio` | 缓存创建倍率 |
| `image_ratio` | 图片倍率 |
| `audio_ratio` | 音频输入倍率 |
| `audio_completion_ratio` | 音频输出倍率 |
| `quota_type` | 计费类型 |
| `supported_endpoint_types` | 支持的端点类型（chat / image / audio / embedding 等） |

---

### 获取倍率配置
```
GET /api/ratio_config
```

> 需在后台「运营设置 → 倍率设置」中开启「暴露倍率配置接口」。此接口有频率限制。

**响应 data 字段：**
```json
{
  "model_ratio": { "gpt-4o": 15.0, "gpt-4o-mini": 0.3 },
  "completion_ratio": { "gpt-4o": 2.0 },
  "cache_ratio": { "gpt-4o": 0.5 },
  "create_cache_ratio": { "gpt-4o": 1.0 },
  "model_price": { "o1-pro": 1.5 }
}
```

---

## 4. 下游对接建议

### 最小接入路径
1. **拉取提示词** → `GET /api/public/prompts` + `GET /api/public/prompt-categories`
2. **拉取文章** → `GET /api/public/articles` + `GET /api/public/article-categories`
3. **拉取模型+价格** → `GET /api/pricing`

### 缓存建议
- `/api/pricing` 数据变化频率低，建议缓存 1-5 分钟
- `/api/ratio_config` 变化频率更低，建议缓存 5-10 分钟
- 提示词/文章列表根据业务需要决定缓存时间

---

*文档生成时间：2026-05-14*
