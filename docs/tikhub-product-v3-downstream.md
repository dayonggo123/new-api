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

接口已做数据加工，直接返回提取后的标准字段，同时保留上游原始 JSON 在 `data` 中。

```json
{
  "success": true,
  "product_id": "1729556031006938037",
  "title": "The Loaded Tea Shop - 40 Flavors, 1 Packet Makes 32 oz...",
  "description": "The Loaded Tea Shop is the ONLY...\nOur loaded teas are made...\nBuild your own bundle...",
  "images": [
    "https://p16-oec-general-useast5.ttcdn-us.com/...webp",
    "https://p16-oec-general-useast5.ttcdn-us.com/...webp"
  ],
  "sold_count": 679027,
  "rating": 4.6,
  "review_count": 31623,
  "shop_name": "The Loaded Tea Shop",
  "shop_logo": "https://p16-oec-general-useast5.ttcdn-us.com/...png",
  "sale_properties": [
    {
      "property_id": "7556608614727812919",
      "property_name": "Flavor",
      "has_image": true,
      "values": [
        { "property_value_id": "7107962144494421765", "property_value_name": "Bahama Mama", "image_url": "https://..." }
      ]
    },
    {
      "property_id": "7627069605563090702",
      "property_name": "Type",
      "has_image": false,
      "values": [
        { "property_value_id": "7176544070252201734", "property_value_name": "Natural Caffeine" },
        { "property_value_id": "7132027973078796038", "property_value_name": "Caffeine Free" }
      ]
    }
  ],
  "product_properties": [
    { "property_id": "101395", "property_name": "CA Prop 65: Repro. Chems", "has_image": false, "values": [{ "property_value_id": "1000059", "property_value_name": "No" }] }
  ],
  "skus": [
    {
      "sku_name": "BAMA1",
      "sku_status": 1,
      "stock_status": 1,
      "available_quantity": 87409,
      "properties": { "Flavor": "Bahama Mama", "Type": "Natural Caffeine" },
      "package_weight": 10,
      "package_length": 6,
      "package_width": 11,
      "package_height": 16
    }
  ],
  "package": {
    "weight": 10,
    "length": 6,
    "width": 11,
    "height": 16
  },
  "data": { ... 上游原始完整 JSON ... }
}
```

## 返回字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | bool | 是否成功 |
| `product_id` | string | 商品 ID |
| `title` | string | 商品标题 |
| `description` | string | 商品描述，多段用 `\n` 连接 |
| `images` | []string | 主图 URL 列表 |
| `sold_count` | int64 | 累计销量 |
| `rating` | float64 | 商品评分 |
| `review_count` | int64 | 评论数 |
| `shop_name` | string | 店铺名称 |
| `shop_logo` | string | 店铺 Logo URL |
| `sale_properties` | []Property | 销售属性（Flavor、Type 等） |
| `product_properties` | []Property | 商品属性（无糖、素食、产地等） |
| `skus` | []SKU | SKU 列表 |
| `package` | object | 默认包装尺寸/重量（取第一个 SKU） |
| `data` | object | 上游 TikHub 原始完整响应 |

### Property 结构（`sale_properties` / `product_properties`）

```json
{
  "property_id": "7556608614727812919",
  "property_name": "Flavor",
  "has_image": true,
  "values": [
    {
      "property_value_id": "7107962144494421765",
      "property_value_name": "Bahama Mama",
      "image_url": "https://..."
    }
  ]
}
```

### SKU 结构

```json
{
  "sku_name": "BAMA1",
  "sku_status": 1,
  "stock_status": 1,
  "available_quantity": 87409,
  "properties": {
    "Flavor": "Bahama Mama",
    "Type": "Natural Caffeine"
  },
  "package_weight": 10,
  "package_length": 6,
  "package_width": 11,
  "package_height": 16
}
```

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
4. 标准字段已做提取；如需更底层数据，直接取 `data` 字段解析上游原始 JSON。
