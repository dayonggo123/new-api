# 提示词库 API 接口文档

> 面向下游应用的公开接口

---

## 一、Prompt 提示词接口

### 1. 获取提示词列表

```
GET /api/public/prompts
```

**认证：** 无需认证（公开接口）

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| num | int | 否 | 每页数量，默认 12 |
| keyword | string | 否 | 搜索关键词（只搜索 title） |
| category_id | int | 否 | 分类 ID 过滤 |
| lang | string | 否 | 语言代码（zh/en/ja/fr/ru/vi 等） |

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "page": 1,
    "num": 12,
    "total": 128,
    "start_idx": 0,
    "items": [
      {
        "id": 1,
        "category_id": 3,
        "title": "小红书文案生成器",
        "description": "自动生成小红书风格的种草文案",
        "cover_image_url": "https://...",
        "video_url": "",
        "author": "@dayong",
        "source": "opennana",
        "model": "ChatGPT",
        "media_type": "image",
        "is_premium": false,
        "unlock_cost": 0,
        "sort_order": 10,
        "status": 1,
        "usage_count": 1523,
        "created_time": 1759300000,
        "updated_time": 1759390000,
        "category_name": "写作助手"
      }
    ]
  }
}
```

> **注意：** 列表接口为性能考虑，**不返回** `content`、`variables`、`tags`、`content_en`、`i18n` 等大字段。详情请调详情接口。

---

### 2. 获取提示词详情

```
GET /api/public/prompts/:id
```

**认证：** 无需认证

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 提示词 ID（路径参数） |
| lang | string | 否 | 语言代码 |

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "category_id": 3,
    "title": "小红书文案生成器",
    "content": "你是一个小红书文案专家，请根据以下信息生成文案...",
    "content_en": "You are a Xiaohongshu copywriting expert...",
    "description": "自动生成小红书风格的种草文案",
    "cover_image_url": "https://...",
    "video_url": "",
    "author": "@dayong",
    "source": "opennana",
    "model": "ChatGPT",
    "variables": "[{\"name\":\"product\",\"label\":\"产品名称\",\"required\":true}]",
    "tags": "[\"文案\",\"小红书\",\"种草\"]",
    "media_type": "image",
    "is_premium": false,
    "unlock_cost": 0,
    "sort_order": 10,
    "status": 1,
    "usage_count": 1524,
    "seo_keywords": "小红书文案,种草文案",
    "intro": "这是一个专业的文案生成工具...",
    "faq": "[{\"q\":\"如何使用？\",\"a\":\"输入产品名称即可\"}]",
    "i18n": "{\"en\":{\"title\":\"Xiaohongshu Copywriter\"}}",
    "created_time": 1759300000,
    "updated_time": 1759390000,
    "category_name": "写作助手"
  }
}
```

> 调用此接口会自动增加该提示词的 `usage_count`（异步统计）。

---

### 3. 获取提示词分类列表

```
GET /api/public/prompt-categories
```

**认证：** 无需认证

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": [
    {
      "id": 1,
      "name": "写作助手",
      "sort_order": 1,
      "status": 1,
      "created_time": 1759300000,
      "updated_time": 1759300000
    }
  ]
}
```

---

### 4. 获取提示词媒体文件

```
GET /api/public/prompt-media/:id
```

**认证：** 无需认证

用于获取提示词相关的图片/视频等媒体资源。

---

## 二、Preset Prompt 预设提示词接口

### 1. 获取预设提示词列表

```
GET /api/public/preset-prompts
```

**认证：** `TryUserAuth`（可选认证，未登录也能访问）

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| lang | string | 否 | 语言代码 |

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": [
    {
      "id": 1,
      "name": "小红书文案生成",
      "system_prompt": "你是一个小红书文案专家...",
      "user_prompt": "请为{{product}}生成一篇小红书文案",
      "description": "自动生成小红书种草文案",
      "category": "写作",
      "status": 1,
      "sort_order": 10,
      "created_time": 1759300000,
      "updated_time": 1759390000
    }
  ]
}
```

> 支持多语言：传入 `?lang=en` 会自动根据 `i18n` 字段替换 `name`、`system_prompt`、`user_prompt`、`description`、`category`。

---

### 2. 获取预设提示词增量更新

```
GET /api/public/preset-prompts/updates
```

**认证：** `TryUserAuth`

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| since | int64 | 否 | 秒级时间戳，只返回该时间之后有更新的提示词 |
| lang | string | 否 | 语言代码 |

**响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "小红书文案生成",
        "category": "写作",
        "status": 1,
        "sort_order": 10,
        "updated_time": 1759390000,
        "created_time": 1759300000
      }
    ],
    "total": 1,
    "server_time": 1759390751
  }
}
```

> **下游使用方式：**
> 1. 首次调用不带 `since`，获取全部
> 2. 记录返回的 `server_time`
> 3. 后续轮询带 `since={上次 server_time}`，只获取变化的数据

---

## 三、字段说明

### Prompt 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 唯一标识 |
| category_id | int | 分类 ID |
| title | string | 标题 |
| content | string | 提示词模板内容（详情接口才有） |
| content_en | string | 英文内容（详情接口才有） |
| description | string | 描述 |
| cover_image_url | string | 封面图 URL |
| video_url | string | 视频 URL |
| author | string | 作者，如 @username |
| source | string | 来源平台，如 opennana / tiktok |
| model | string | AI 模型，如 ChatGPT |
| variables | string | 变量定义 JSON 数组（详情接口才有） |
| tags | string | 标签 JSON 数组（详情接口才有） |
| media_type | string | image / video |
| is_premium | bool | 是否为付费提示词 |
| unlock_cost | int | 解锁所需积分 |
| sort_order | int | 排序权重 |
| status | int | 1=启用, 2=禁用 |
| usage_count | int | 使用次数 |
| seo_keywords | string | SEO 关键词（详情接口才有） |
| intro | string | 介绍文案（详情接口才有） |
| faq | string | FAQ JSON（详情接口才有） |
| i18n | string | 多语言 JSON（详情接口才有） |
| created_time | int64 | 创建时间（秒级时间戳） |
| updated_time | int64 | 更新时间（秒级时间戳） |
| category_name | string | 分类名称（关联查询） |

### PresetPrompt 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 唯一标识 |
| name | string | 名称 |
| system_prompt | string | 系统提示词 |
| user_prompt | string | 用户提示词模板 |
| description | string | 描述 |
| category | string | 分类 |
| status | int | 1=启用, 2=禁用 |
| sort_order | int | 排序权重 |
| created_time | int64 | 创建时间 |
| updated_time | int64 | 更新时间 |

---

## 四、调用示例

### 获取提示词列表

```bash
curl "https://heharse.cloud/api/public/prompts?page=1&num=12&category_id=3"
```

### 获取提示词详情

```bash
curl "https://heharse.cloud/api/public/prompts/1"
```

### 获取预设提示词（英文）

```bash
curl "https://heharse.cloud/api/public/preset-prompts?lang=en" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### 增量同步预设提示词

```bash
# 首次同步
curl "https://heharse.cloud/api/public/preset-prompts/updates"

# 后续增量同步（假设上次 server_time = 1759390000）
curl "https://heharse.cloud/api/public/preset-prompts/updates?since=1759390000"
```

---

## 五、通用响应格式

所有接口统一返回以下格式：

```json
{
  "success": true,
  "message": "",
  "data": { ... }
}
```

错误时：

```json
{
  "success": false,
  "message": "错误描述",
  "data": null
}
```
