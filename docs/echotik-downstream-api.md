# EchoTik 视频榜单接口 — 下游调用文档

## 接口地址

```
GET https://你的new-api域名/api/public/echotik/video/ranklist
```

## 鉴权方式

在请求头中传入 new-api 的 API Key：

```
Authorization: Bearer sk-xxxxxx
```

## 请求参数

参数原样透传给 EchoTik：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `date` | string | 是 | 榜单日期，格式 `yyyy-MM-dd` |
| `region` | string | 是 | 区域代码，例如 `US` |
| `video_rank_field` | integer | 是 | 排序字段：`1`=热门榜，`2`=带货榜 |
| `rank_type` | integer | 是 | 榜单类型：`1`=日榜，`2`=周榜，`3`=月榜 |
| `page_num` | integer | 是 | 页码，从 `1` 开始，最大 `100000` |
| `page_size` | integer | 是 | 每页条数，最大 `10` |
| `product_category_id` | string | 否 | 商品一级类目 ID，用于带货榜筛选 |
| `created_by_ai` | string | 否 | `true` / `false`，是否 AI 视频 |

## 请求示例

```bash
curl -X GET 'https://你的new-api域名/api/public/echotik/video/ranklist?date=2026-06-29&region=US&video_rank_field=1&rank_type=1&page_num=1&page_size=10' \
  --header 'Authorization: Bearer sk-xxxxxx'
```

## 响应示例

new-api 直接透传 EchoTik 原始响应：

```json
{
  "code": 200,
  "message": "success",
  "requestId": "xxx",
  "data": [
    {
      "avatar": "",
      "category": "",
      "create_time": "",
      "duration": 0,
      "nick_name": "",
      "product_category_list": "",
      "reflow_cover": "",
      "region": "US",
      "sales_flag": 0,
      "total_comments_cnt": 0,
      "total_digg_cnt": 0,
      "total_favorites_cnt": 0,
      "total_shares_cnt": 0,
      "total_video_sale_cnt": 0,
      "total_video_sale_gmv_amt": 0,
      "total_views_cnt": 0,
      "unique_id": "",
      "user_id": "",
      "video_desc": "",
      "video_id": "",
      "video_products": "",
      "created_by_ai": ""
    }
  ]
}
```

## 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `avatar` | string | 达人头像封面地址 |
| `category` | string | 达人分类 |
| `create_time` | string | 视频发布时间 |
| `duration` | integer | 视频时长 |
| `nick_name` | string | 昵称 |
| `product_category_list` | string | 带货商品分类信息 |
| `reflow_cover` | string | 视频封面地址 |
| `region` | string | 地区 |
| `sales_flag` | integer | 主要带货方式：`0`=不带货，`1`=视频带货，`2`=直播带货 |
| `total_comments_cnt` | integer | 总评论量 |
| `total_digg_cnt` | integer | 总点赞量 |
| `total_favorites_cnt` | integer | 总收藏量 |
| `total_shares_cnt` | integer | 总分享量 |
| `total_video_sale_cnt` | integer | 视频销量（预估） |
| `total_video_sale_gmv_amt` | integer | 视频销售额（预估） |
| `total_views_cnt` | integer | 总播放量 |
| `unique_id` | string | TikTok ID |
| `user_id` | string | 达人 ID |
| `video_desc` | string | 视频描述 |
| `video_id` | string | 视频 ID |
| `video_products` | string | 视频带货商品信息 |
| `created_by_ai` | string | 是否 AI 视频 |

## 注意事项

1. `date` 字段格式必须为 `yyyy-MM-dd`
2. 周榜日期请传每周周一，月榜日期请传每月一号
3. 榜单中返回的数值是当前周期内的增量数据
4. 所有 API 请求必须携带有效的 `Authorization` 头，否则返回 `401` 未授权
5. 如需调整每页条数上限或其他限制，请联系 new-api 管理员
