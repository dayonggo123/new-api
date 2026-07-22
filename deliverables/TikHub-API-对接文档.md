# TikHub API 接口对接文档

> 更新时间：2026-07-22

---

## 基础信息

| 项目 | 说明 |
|------|------|
| Base URL | `https://你的域名.com/api` |
| 鉴权方式 | Header: `Authorization: Bearer <你的API Token>` |
| 数据格式 | JSON |
| 字符编码 | UTF-8 |

---

## 接口列表

### 1. 通过分享链接获取视频

获取 TikTok 视频数据（通过分享链接）

```http
GET /api/public/tikhub/tiktok/video-by-share-url
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| share_url | string | 是 | TikTok 分享链接 |

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/video-by-share-url?share_url=https://www.tiktok.com/@user/video/7645541495902129439" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 2. 评论关键词分析

分析视频评论中的热门关键词

```http
GET /api/public/tikhub/tiktok/comment-keywords
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| item_id | string | 是 | 视频作品ID |

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/comment-keywords?item_id=7502551047378832671" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 3. 音乐排行榜

获取 TikTok 热门音乐排行榜

```http
GET /api/public/tikhub/tiktok/music-chart-list
```

**参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| scene | int | 否 | 0 | 排行榜类型: 0=Top 50, 1=Viral 50 |
| cursor | int | 否 | 0 | 分页游标 |
| count | int | 否 | 50 | 每页数量，最大50 |

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/music-chart-list?scene=0&count=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 4. 每日趋势搜索关键词

获取每日趋势搜索关键词

```http
GET /api/public/tikhub/tiktok/trending-search-words
```

**参数：** 无

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/trending-search-words" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 5. 创作者账号健康状态

获取 TikTok Shop 创作者账号健康状况（违规积分）

```http
POST /api/public/tikhub/tiktok/account-health-status
Content-Type: application/json
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | 用户 Cookie 字符串 |
| proxy | string | 否 | HTTP 代理地址，格式: `http://username:password@host:port` |

**请求体示例：**
```json
{
  "cookie": "your_cookie_string",
  "proxy": ""
}
```

**返回字段说明：**

| 字段 | 说明 |
|------|------|
| risk_info.risk_level_text | 风险等级描述 (如 Healthy) |
| risk_info.light_color | 浅色展示色值 |
| risk_info.dark_color | 深色展示色值 |
| is_show_score | 是否显示违规积分 |
| violation_score | 违规积分数量 |
| creator_status | 账号状态码 (0=正常) |

---

### 6. 创作者账号概览

获取创作者账号在指定时间范围内的表现概览

```http
POST /api/public/tikhub/tiktok/account-insights-overview
Content-Type: application/json
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | 用户 Cookie 字符串 |
| start_date | string | 否 | 查询开始时间，格式: `MM-DD-YYYY`，默认 04-01-2025 |
| proxy | string | 否 | HTTP 代理地址 |

**请求体示例：**
```json
{
  "cookie": "your_cookie_string",
  "start_date": "04-01-2025",
  "proxy": ""
}
```

**返回字段说明：**

| 字段 | 说明 |
|------|------|
| segments[].timed_stats[].live_revenue.amount | 直播带货收益 |
| segments[].timed_stats[].video_revenue.amount | 视频带货收益 |
| segments[].timed_stats[].revenue.amount | 总收益 |
| segments[].timed_stats[].overall_item_sold_cnt | 商品成交数 |
| segments[].timed_stats[].product_show_cnt | 商品展示次数 |
| segments[].timed_stats[].product_click_cnt | 商品点击次数 |

---

### 7. 创作者视频概览

获取创作者视频表现概览

```http
POST /api/public/tikhub/tiktok/video-analytics-summary
Content-Type: application/json
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | 用户 Cookie 字符串 |
| proxy | string | 否 | HTTP 代理地址 |

**请求体示例：**
```json
{
  "cookie": "your_cookie_string"
}
```

---

### 8. 同款商品关联视频

获取与指定商品关联的所有视频列表

```http
POST /api/public/tikhub/tiktok/product-related-videos
Content-Type: application/json
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cookie | string | 是 | 用户 Cookie 字符串 |
| item_id | string | 是 | 当前视频 ID |
| product_id | string | 是 | 商品 ID |
| start_date | string | 否 | 查询开始时间，格式: `MM-DD-YYYY` |
| proxy | string | 否 | HTTP 代理地址 |

**请求体示例：**
```json
{
  "cookie": "your_cookie_string",
  "item_id": "7496499484705246507",
  "product_id": "1731050202505515549",
  "start_date": "04-01-2025"
}
```

---

### 9. 热门标签榜单

获取 TikTok Creative Center 趋势板块的热门标签

```http
GET /api/public/tikhub/tiktok/trends-hashtag-list
```

**参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| time_range | int | 否 | 7 | 时间范围(天): 7 / 30 / 90 |
| country_code | string | 否 | US | 国家代码: US/GB/SG/MY/PH/TH/VN/ID |
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 20 | 每页数量 |
| industry_id | int64 | 否 | - | 行业ID (可选) |

**行业ID参考：**
- 10000000000: 教育
- 11000000000: 汽车与交通
- 12000000000: 母婴
- 13000000000: 金融服务
- 14000000000: 美妆个护
- 15000000000: 科技与电子
- 22000000000: 服装配饰
- 27000000000: 食品饮料

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/trends-hashtag-list?time_range=7&country_code=US" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

### 10. 热卖商品列表

获取 TikTok Shop 热卖商品列表

```http
GET /api/public/tikhub/tiktok/hot-selling-products-list
```

**参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| region | string | 否 | US | 地区代码: US/GB/SG/MY/PH/TH/VN/ID |
| count | int | 否 | 100 | 返回商品数量 |

**示例：**
```bash
curl -X GET "https://你的域名.com/api/public/tikhub/tiktok/hot-selling-products-list?region=US&count=50" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 响应格式

所有接口返回 JSON 格式数据，透传上游 TikHub 原始响应。

**成功响应示例：**
```json
{
  "code": 200,
  "message": "Request successful. This request will incur a charge.",
  "data": {
    // 业务数据
  }
}
```

**错误响应示例：**
```json
{
  "success": false,
  "message": "TikHub API Key 未配置"
}
```

---

## 错误码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 鉴权失败 |
| 422 | 参数验证失败 |
| 502 | 上游服务错误 |
| 503 | TikHub 接口未启用 |

---

## 注意事项

1. **计费**：调用 TikHub 接口会产生费用，具体请参考 TikHub 官方定价
2. **Cookie**：部分接口需要用户 Cookie，请确保 Cookie 有效且具有相应权限
3. **代理**：如需使用代理，请确保代理服务器稳定
4. **频率限制**：请合理控制请求频率，避免触发 TikHub 限流
