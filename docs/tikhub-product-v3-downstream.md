# TikHub TikTok 商品详情 V3 下游对接文档

## 对接接口

```http
GET /api/public/tikhub/tiktok/product-v3?product_id={product_id}&region={region}
```

## 上游 TikHub 接口

```text
GET /api/v1/tiktok/shop/web/fetch_product_detail_v3?product_id=xxx&region=xxx
```

## 请求方式

- 方法：`GET`
- 鉴权：`Authorization: Bearer {your_new_api_token}`
- Content-Type：`application/json`

## Query 参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| product_id | string | 是 | - | TikTok 商品 ID |
| region | string | 否 | SG | 地区代码：`US` / `GB` / `SG` / `MY` / `PH` / `TH` / `VN` / `ID` |

## 示例请求

```bash
curl -G "https://heharse.cloud/api/public/tikhub/tiktok/product-v3" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Accept: application/json" \
  --data-urlencode "product_id=1729556031006938037" \
  --data-urlencode "region=US"
```

## 示例响应

```json
{
  "code": 200,
  "request_id": "...",
  "message": "Request successful. This request will incur a charge.",
  "time": "2026-07-30 04:55:43",
  "router": "/api/v1/tiktok/shop/web/fetch_product_detail_v3",
  "params": {
    "product_id": "1729556031006938037",
    "region": "US"
  },
  "data": {
    "product_data": {
      "ab_test": { ... },
      "page_config": {
        "components_map": [
          { "component_type": "user_info", ... },
          { "component_type": "product_info", "component_data": { ... } },
          { "component_type": "related_videos", ... },
          ...
        ],
        "global_data": { ... }
      }
    }
  }
}
```

## 可提取的商品字段

实际商品详情位于 `data.product_data.page_config.components_map` 中 `component_type=product_info` 的组件内：

| 字段 | 路径 | 说明 |
|------|------|------|
| 商品 ID | `...product_info.product_info.product_model.product_id` | - |
| 商品名称 | `...product_info.product_info.product_model.name` | 完整标题 |
| 商品描述 | `...product_info.product_info.product_model.description` | 富文本数组，含 `sub[].t` |
| 累计销量 | `...product_info.product_info.product_model.sold_count` | 数字 |
| 主图 | `...product_info.product_info.product_model.images[].url_list[0]` | 高清图 URL |
| SKU 列表 | `...product_info.product_info.product_model.skus[]` | 80 个 SKU，含库存、规格组合 |
| 销售属性 | `...product_info.product_info.product_model.sale_properties[]` | 如 Flavor、Type |
| 商品属性 | `...product_info.product_info.product_model.product_properties[]` | 21 项属性 |
| 评分 | `...product_info.review_model.product_overall_score` | 如 `4.6` |
| 评论数 | `...product_info.review_model.product_review_count` | 如 `31623` |
| 店铺名称 | `...product_info.seller_model.shop_name` | - |
| 店铺 Logo | `...product_info.seller_model.shop_logo.url_list[0]` | - |

## 价格说明

当前 TikHub V3 公开接口对实际价格做了脱敏处理，价格字段显示为 `*`：

```json
{
  "sale_price_decimal": "*",
  "origin_price_decimal": "*",
  "discount_format": "50%",
  "reduce_price_format": "Saving $*"
}
```

只能看到折扣比例，无法拿到真实售价/原价。

## 计费

调用本接口会按 TikHub 配置的价格扣费。需在后台 `运营设置 → TikHub 价格设置` 中配置 `product-v3` 积分。

## 注意事项

1. **region 必须匹配商品实际地区**，否则可能返回空数据。
2. 请求超时设置为 30 秒。
3. 如遇 400 错误，可重试 3 次。
4. 返回的是 TikHub 上游原始 JSON，如需标准化字段，需自行解析 `components_map`。
