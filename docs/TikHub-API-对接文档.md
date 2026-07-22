# TikHub API 对接文档

## 基础信息

- **API Base URL**: `https://heharse.cloud/api`
- **鉴权方式**: Bearer Token
- **调用示例**:

```bash
# 获取视频评论
curl -X GET "https://heharse.cloud/api/public/tikhub/tiktok/video-comments?aweme_id=xxx" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 获取视频受众分析
curl -X POST "https://heharse.cloud/api/public/tikhub/tiktok/video-audience-stats" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cookie": "xxx", "item_id": "xxx"}'
```

---

## 价格查询接口

### 获取接口价格列表

获取所有接口的当前价格配置（返回已启用的配置）：

```bash
curl -X GET "https://heharse.cloud/api/public/tikhub/prices"
```

**响应示例**：

```json
{
  "success": true,
  "data": [
    {
      "endpoint": "video",
      "name": "获取单个视频数据",
      "price": 0.01,
      "enabled": true,
      "quota": 1,
      "description": "通过 aweme_id 获取视频数据"
    },
    {
      "endpoint": "video-comments",
      "name": "获取视频评论",
      "price": 0.02,
      "enabled": true,
      "quota": 2,
      "description": "获取单个视频评论数据"
    }
  ]
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| endpoint | string | 接口标识 |
| name | string | 接口名称 |
| price | float | 价格（USD） |
| enabled | bool | 是否启用 |
| quota | int | 消耗积分（= price × 100） |
| description | string | 接口描述 |

---

## 完整接口列表

| 接口 | 地址 | 方法 | 说明 |
|------|------|------|------|
| 获取单个视频数据 | `/api/public/tikhub/tiktok/video` | GET | 通过 aweme_id 获取 |
| 通过分享链接获取视频 | `/api/public/tikhub/tiktok/video-by-share-url` | GET | 通过分享链接获取 |
| 获取视频评论 | `/api/public/tikhub/tiktok/video-comments` | GET | 获取视频评论列表 |
| 获取作品评论列表 | `/api/public/tikhub/tiktok/post-comment` | GET | 获取作品评论列表 |
| 评论关键词分析 | `/api/public/tikhub/tiktok/comment-keywords` | GET | 评论关键词分析 |
| 音乐排行榜 | `/api/public/tikhub/tiktok/music-chart-list` | GET | TikTok 音乐排行榜 |
| 每日趋势搜索词 | `/api/public/tikhub/tiktok/trending-search-words` | GET | 每日趋势搜索关键词 |
| 商品详情 | `/api/public/tikhub/tiktok/product` | GET | TikTok 商品详情 |
| 视频受众分析 | `/api/public/tikhub/tiktok/video-audience-stats` | POST | 视频受众分析数据 |
| 账号健康状态 | `/api/public/tikhub/tiktok/account-health-status` | POST | 创作者账号健康状态 |
| 账号概览 | `/api/public/tikhub/tiktok/account-insights-overview` | POST | 创作者账号概览 |
| 视频概览 | `/api/public/tikhub/tiktok/video-analytics-summary` | POST | 创作者视频概览 |
| 同款商品关联视频 | `/api/public/tikhub/tiktok/product-related-videos` | POST | 同款商品关联视频 |
| 热门标签榜单 | `/api/public/tikhub/tiktok/trends-hashtag-list` | GET | 热门标签榜单 |
| 热卖商品列表 | `/api/public/tikhub/tiktok/hot-selling-products-list` | GET | 热卖商品列表 |

---

## 接口详情

### 1. 获取单个视频数据

```http
GET /api/public/tikhub/tiktok/video?aweme_id=7350810998023949599
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| aweme_id | string | 是 | TikTok 作品 ID |

---

### 2. 通过分享链接获取视频

```http
GET /api/public/tikhub/tiktok/video-by-share-url?share_url=https://www.tiktok.com/@xxx/video/xxx
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| share_url | string | 是 | TikTok 分享链接 |

---

### 3. 获取视频评论

```http
GET /api/public/tikhub/tiktok/video-comments?aweme_id=xxx&cursor=0&count=20
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| aweme_id | string | 是 | TikTok 作品 ID |
| cursor | int | 否 | 分页游标，默认 0 |
| count | int | 否 | 数量，默认 20 |

---

### 4. 获取作品评论列表

```http
GET /api/public/tikhub/tiktok/post-comment?aweme_id=xxx&cursor=0&count=20
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| aweme_id | string | 是 | TikTok 作品 ID |
| cursor | int | 否 | 分页游标，默认 0 |
| count | int | 否 | 数量，默认 20 |
| current_region | string | 否 | 当前地区 |

---

### 5. 评论关键词分析

```http
GET /api/public/tikhub/tiktok/comment-keywords?item_id=7502551047378832671
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| item_id | string | 是 | TikTok 作品 ID |

---

### 6. 音乐排行榜

```http
GET /api/public/tikhub/tiktok/music-chart-list?scene=0&cursor=0&count=50
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| scene | int | 否 | 场景：0=Top 50, 1=Viral 50，默认 0 |
| cursor | int | 否 | 分页游标，默认 0 |
| count | int | 否 | 数量，最大 50，默认 50 |

---

### 7. 每日趋势搜索词

```http
GET /api/public/tikhub/tiktok/trending-search-words
```

无需参数。

---

### 8. 商品详情

```http
GET /api/public/tikhub/tiktok/product?product_id=1729385239712731370
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| product_id | string | 是 | TikTok 商品 ID |

---

### 9. 视频受众分析

```http
POST /api/public/tikhub/tiktok/video-audience-stats
Content-Type: application/json
```

**请求体**:
```json
{
  "cookie": "xxx",
  "item_id": "xxx",
  "start_date": "04-01-2025",
  "proxy": "xxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | TikTok cookie |
| item_id | string | 是 | 视频 ID |
| start_date | string | 否 | 开始日期，格式 MM-DD-YYYY，默认 04-01-2025 |
| proxy | string | 否 | 代理地址 |

---

### 10. 账号健康状态

```http
POST /api/public/tikhub/tiktok/account-health-status
Content-Type: application/json
```

**请求体**:
```json
{
  "cookie": "xxx",
  "proxy": "xxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | TikTok cookie |
| proxy | string | 否 | 代理地址 |

---

### 11. 账号概览

```http
POST /api/public/tikhub/tiktok/account-insights-overview
Content-Type: application/json
```

**请求体**:
```json
{
  "cookie": "xxx",
  "start_date": "04-01-2025",
  "proxy": "xxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | TikTok cookie |
| start_date | string | 否 | 开始日期，格式 MM-DD-YYYY，默认 04-01-2025 |
| proxy | string | 否 | 代理地址 |

---

### 12. 视频概览

```http
POST /api/public/tikhub/tiktok/video-analytics-summary
Content-Type: application/json
```

**请求体**:
```json
{
  "cookie": "xxx",
  "proxy": "xxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | TikTok cookie |
| proxy | string | 否 | 代理地址 |

---

### 13. 同款商品关联视频

```http
POST /api/public/tikhub/tiktok/product-related-videos
Content-Type: application/json
```

**请求体**:
```json
{
  "cookie": "xxx",
  "item_id": "xxx",
  "product_id": "xxx",
  "start_date": "04-01-2025",
  "proxy": "xxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | TikTok cookie |
| item_id | string | 是 | 视频 ID |
| product_id | string | 是 | 商品 ID |
| start_date | string | 否 | 开始日期，格式 MM-DD-YYYY，默认 04-01-2025 |
| proxy | string | 否 | 代理地址 |

---

### 14. 热门标签榜单

```http
GET /api/public/tikhub/tiktok/trends-hashtag-list?time_range=7&country_code=US&page=1&limit=20
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| time_range | int | 否 | 时间范围：7/30/90 天，默认 7 |
| country_code | string | 否 | 国家代码，默认 US |
| page | int | 否 | 页码，默认 1 |
| limit | int | 否 | 每页数量，默认 20 |
| industry_id | int64 | 否 | 行业 ID |

---

### 15. 热卖商品列表

```http
GET /api/public/tikhub/tiktok/hot-selling-products-list?region=US&count=100
```

**Query 参数**:
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| region | string | 否 | 地区代码，默认 US |
| count | int | 否 | 返回商品数量，默认 100 |

---

## 相关文件

- 后端接口实现：`controller/tikhub.go`
- 上游转发逻辑：`service/tikhub.go`
- 配置定义：`setting/operation_setting/tikhub_setting.go`
- 路由注册：`router/api-router.go`
