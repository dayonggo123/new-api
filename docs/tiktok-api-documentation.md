# TikHub API 对接文档

> 版本: 1.0.0 | 更新日期: 2026-07-23

## 基础信息

- **Base URL**: `https://your-domain.com/api`
- **认证方式**: Bearer Token (Header: `Authorization: Bearer {token}`)
- **价格单位**: 美元 (USD)，实际扣费 = 价格 × 100 积分

---

## 接口列表

### 一、获取接口价格配置

获取所有 TikHub 接口的价格配置列表。

| 项目 | 说明 |
|------|------|
| **接口** | `GET /api/public/tikhub/prices` |
| **认证** | 无需认证 |
| **返回** | 所有启用的接口配置 |

**返回示例**:
```json
{
  "success": true,
  "data": [
    {
      "endpoint": "video",
      "name": "获取单个视频数据",
      "price": 0.01,
      "enabled": true,
      "category": "video",
      "requires_cookie": false
    }
  ]
}
```

---

## 二、接口分类列表

### 📹 视频类 (Video)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| video | 获取单个视频数据 | `/api/public/tikhub/tiktok/video` | GET | $0.01 | ❌ | 通过 aweme_id 获取视频数据 |
| video-by-share-url | 通过分享链接获取视频 | `/api/public/tikhub/tiktok/video-by-share-url` | GET | $0.01 | ❌ | 通过分享链接获取视频数据 |
| video-comments | 获取视频评论 | `/api/public/tikhub/tiktok/video-comments` | GET | $0.02 | ❌ | 获取单个视频评论数据 |
| post-comment | 获取作品评论列表 | `/api/public/tikhub/tiktok/post-comment` | GET | $0.02 | ❌ | 获取作品评论列表(Web) |
| video-metrics | 视频统计数据 | `/api/public/tikhub/tiktok/video-metrics` | GET | $0.03 | ❌ | 获取视频观看量、点赞、评论、收藏等 |
| detect-fake-views | 虚假流量检测 | `/api/public/tikhub/tiktok/detect-fake-views` | GET | $0.05 | ❌ | 检测视频虚假流量分析 |
| video-audience-stats | 视频受众分析 | `/api/public/tikhub/tiktok/video-audience-stats` | POST | $0.03 | ✅ | 获取视频受众分析数据 |

**公共参数**:
- `aweme_id`: 视频ID (必填)

---

### 🔍 搜索类 (Search)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| general-search-result | 综合搜索 | `/api/public/tikhub/tiktok/general-search-result` | GET | $0.02 | ❌ | 获取指定关键词的综合搜索结果 |
| video-search-result | 视频搜索 | `/api/public/tikhub/tiktok/video-search-result` | GET | $0.02 | ❌ | 获取指定关键词的视频搜索结果 |
| user-search-result | 用户搜索 | `/api/public/tikhub/tiktok/user-search-result` | GET | $0.02 | ❌ | 获取指定关键词的用户搜索结果 |
| music-search-result | 音乐搜索 | `/api/public/tikhub/tiktok/music-search-result` | GET | $0.02 | ❌ | 获取指定关键词的音乐搜索结果 |
| hashtag-search-result | 话题搜索 | `/api/public/tikhub/tiktok/hashtag-search-result` | GET | $0.02 | ❌ | 获取指定关键词的话题搜索结果 |
| trending-search-words | 每日趋势搜索词 | `/api/public/tikhub/tiktok/trending-search-words` | GET | $0.01 | ❌ | 获取每日趋势搜索关键词 |
| user-country-by-username | 用户国家地区 | `/api/public/tikhub/tiktok/user-country-by-username` | GET | $0.01 | ❌ | 通过用户名获取用户账号国家地区 |

---

### 🎵 音乐类 (Music)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| music-detail | 音乐详情 | `/api/public/tikhub/tiktok/music-detail` | GET | $0.02 | ❌ | 获取指定音乐的详情数据 |
| music-video-list | 音乐视频列表 | `/api/public/tikhub/tiktok/music-video-list` | GET | $0.02 | ❌ | 获取指定音乐的视频列表数据 |
| music-chart-list | 音乐排行榜 | `/api/public/tikhub/tiktok/music-chart-list` | GET | $0.01 | ❌ | 获取热门音乐排行榜 |

---

### # 话题类 (Hashtag)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| hashtag-detail | 话题详情 | `/api/public/tikhub/tiktok/hashtag-detail` | GET | $0.02 | ❌ | 获取指定话题的详情数据 |
| hashtag-video-list | 话题视频列表 | `/api/public/tikhub/tiktok/hashtag-video-list` | GET | $0.02 | ❌ | 获取指定话题的作品数据 |

---

### 📦 商品类 (Product)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| product | 商品详情 | `/api/public/tikhub/tiktok/product` | GET | $0.02 | ❌ | 获取 TikTok 商品详情 |
| shop-product-detail | 商品详情V1 | `/api/public/tikhub/tiktok/shop-product-detail` | GET | $0.03 | ❌ | 获取TikTok Shop商品详情V1 |
| product-reviews-v1 | 商品评论V1 | `/api/public/tikhub/tiktok/product-reviews` | GET | $0.02 | ❌ | 获取TikTok Shop商品评论V1 |
| product-reviews-v2 | 商品评论V2 | `/api/public/tikhub/tiktok/product-reviews-v2` | GET | $0.02 | ❌ | 获取TikTok Shop商品评论V2 |
| seller-products-list | 商家商品列表 | `/api/public/tikhub/tiktok/seller-products-list` | GET | $0.02 | ❌ | 获取指定商家的商品列表 |
| search-products-list | 搜索商品列表 | `/api/public/tikhub/tiktok/search-products-list` | GET | $0.02 | ❌ | 根据关键词搜索商品列表 |
| hot-selling-products-list-v1 | 热卖商品列表V1 | `/api/public/tikhub/tiktok/hot-selling-products-list-v1` | GET | $0.02 | ❌ | 获取TikTok Shop热卖商品列表 |
| hot-selling-products-list | 热卖商品列表 | `/api/public/tikhub/tiktok/hot-selling-products-list` | GET | $0.01 | ❌ | 获取 TikTok Shop 热卖商品列表 |
| product-related-videos | 同款商品关联视频 | `/api/public/tikhub/tiktok/product-related-videos` | POST | $0.02 | ❌ | 获取同款商品关联视频列表 |
| product-analytics-list | 商品列表分析 | `/api/public/tikhub/tiktok/product-analytics-list` | POST | $0.04 | ✅ | 获取创作者推广商品列表及销售数据 |

---

### 👤 创作者类 (Creator)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| creator-search-insights | 创作者搜索洞察 | `/api/public/tikhub/tiktok/creator-search-insights` | GET | $0.03 | ❌ | 获取创作者搜索洞察数据 |
| creator-search-insights-detail | 创作者搜索洞察详情 | `/api/public/tikhub/tiktok/creator-search-insights-detail` | GET | $0.03 | ❌ | 获取创作者搜索洞察详情数据 |
| creator-search-insights-videos | 创作者搜索洞察视频 | `/api/public/tikhub/tiktok/creator-search-insights-videos` | GET | $0.03 | ❌ | 获取创作者搜索洞察相关视频 |
| creator-info-milestones | 创作者信息与里程碑 | `/api/public/tikhub/tiktok/creator-info-milestones` | GET | $0.03 | ❌ | 获取创作者基本信息和成长里程碑 |
| account-health-status | 账号健康状态 | `/api/public/tikhub/tiktok/account-health-status` | POST | $0.03 | ✅ | 获取创作者账号健康状态 |
| account-insights-overview | 账号概览 | `/api/public/tikhub/tiktok/account-insights-overview` | POST | $0.03 | ✅ | 获取创作者账号表现概览 |
| account-violation-list | 账号违规记录列表 | `/api/public/tikhub/tiktok/account-violation-list` | POST | $0.03 | ✅ | 获取创作者账号违规记录列表 |
| video-analytics-summary | 视频概览 | `/api/public/tikhub/tiktok/video-analytics-summary` | POST | $0.03 | ✅ | 获取创作者视频表现概览 |
| video-list-analytics | 视频列表分析 | `/api/public/tikhub/tiktok/video-list-analytics` | POST | $0.04 | ✅ | 获取创作者视频列表及详细数据 |
| comment-keywords | 评论关键词分析 | `/api/public/tikhub/tiktok/comment-keywords` | GET | $0.02 | ❌ | 分析视频评论中的热门关键词 |

---

### 🔥 趋势类 (Trends)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| trends-hashtag-list | 热门标签榜单 | `/api/public/tikhub/tiktok/trends-hashtag-list` | GET | $0.01 | ❌ | 获取热门标签排行榜 |

---

### 📊 广告类 (Ads)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| ads-search-ads | 搜索广告 | `/api/public/tikhub/tiktok/ads/search-ads` | GET | $0.02 | ❌ | 搜索TikTok广告创意库中的广告 |
| ads-top-ads-spotlight | 热门广告聚光灯 | `/api/public/tikhub/tiktok/ads/top-ads-spotlight` | GET | $0.02 | ❌ | 获取特定行业的热门广告聚光灯 |
| ads-ad-keyframe-analysis | 广告关键帧分析 | `/api/public/tikhub/tiktok/ads/ad-keyframe-analysis` | GET | $0.03 | ❌ | 获取广告视频的关键帧分析 |
| ads-ad-percentile | 广告百分位数据 | `/api/public/tikhub/tiktok/ads/ad-percentile` | GET | $0.03 | ❌ | 获取广告在同行业中的百分位排名数据 |
| ads-ad-interactive-analysis | 广告互动分析 | `/api/public/tikhub/tiktok/ads/ad-interactive-analysis` | GET | $0.03 | ❌ | 获取广告的互动时间分析 |
| ads-trends-hashtag-detail | 热门标签详情 | `/api/public/tikhub/tiktok/ads/trends-hashtag-detail` | GET | $0.02 | ❌ | 获取热门标签的详细数据 |

---

### 📋 整合报告类 (Report)

| 接口标识 | 接口名称 | 路由 | 方法 | 价格 | 需要Cookie | 说明 |
|---------|---------|------|------|------|-----------|------|
| report-product-analysis | 商品分析报告 | `/api/public/tikhub/report/product-analysis` | GET | $0.10 | ❌ | 整合商品详情、评论、关联视频、热卖商品 |
| report-creator-diagnosis | 创作者诊断报告 | `/api/public/tikhub/report/creator-diagnosis` | POST | $0.15 | ✅ | 整合账号概览、健康状态、违规记录、视频分析、受众分析 |
| report-ad-creative-analysis | 广告创意分析报告 | `/api/public/tikhub/report/ad-creative-analysis` | GET | $0.12 | ❌ | 整合热门广告、关键帧、百分位、互动分析 |
| report-content-trends | 内容趋势报告 | `/api/public/tikhub/report/content-trends` | GET | $0.08 | ❌ | 整合热门标签、趋势搜索词、音乐排行、综合搜索 |
| report-video-analysis | 视频深度分析报告 | `/api/public/tikhub/report/video-analysis` | GET | $0.10 | ❌ | 整合视频数据、统计、虚假流量、评论、关键词、受众 |
| report-competitor-monitor | 竞品监控报告 | `/api/public/tikhub/report/competitor-monitor` | GET | $0.08 | ❌ | 整合商家商品、搜索商品、热卖商品 |

---

## 三、接口调用示例

### 1. 获取单个视频数据

```bash
curl -X GET "https://your-domain.com/api/public/tikhub/tiktok/video?aweme_id=123456789" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 2. 搜索商品

```bash
curl -X GET "https://your-domain.com/api/public/tikhub/tiktok/search-products-list?keyword=phone" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 3. 获取商品分析报告

```bash
curl -X GET "https://your-domain.com/api/public/tikhub/report/product-analysis?product_id=xxx&region=US" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 4. 获取创作者诊断报告 (需要Cookie)

```bash
curl -X POST "https://your-domain.com/api/public/tikhub/report/creator-diagnosis" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"cookie": "your_tiktok_cookie"}'
```

---

## 四、错误响应

```json
{
  "success": false,
  "message": "TikHub 接口未启用"
}
```

---

## 五、注意事项

1. **需要 Cookie 的接口**: 部分接口需要用户提供 TikTok 账号 Cookie 才能调用，主要涉及创作者私有数据
2. **扣费说明**: 接口调用时自动扣除积分，管理员用户免费
3. **地区代码**: 部分接口支持地区参数，常用值: US, GB, SG, MY, PH, TH, VN, ID
4. **时间范围**: 部分接口支持时间筛选，常用值: 7, 30, 90 (天)
