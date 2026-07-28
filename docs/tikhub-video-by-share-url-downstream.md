# TikHub 通过分享链接获取单个作品数据 V2 下游对接文档

## 对接接口

```http
GET /api/public/tikhub/tiktok/video-by-share-url?share_url={share_url}
```

## 上游 TikHub 接口

```http
GET /api/v1/tiktok/app/v3/fetch_one_video_by_share_url?share_url={share_url}
```

## 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| `share_url` | query | string | 是 | TikTok 分享链接，例如 `https://www.tiktok.com/@xxx/video/xxx` 或短链 `https://vm.tiktok.com/xxx` |

## 鉴权方式

```http
Authorization: Bearer {new-api-token}
```

使用 new-api 后台生成的 **TokenKey**，调用时需要该 Token 有 TikHub 接口的权限。

## 请求示例

```bash
curl -H "Authorization: Bearer your_newapi_token" \
  "https://heharse.cloud/api/public/tikhub/tiktok/video-by-share-url?share_url=https%3A%2F%2Fwww.tiktok.com%2F%40boise_brooke%2Fvideo%2F7665796670227008781"
```

## 响应示例

new-api 已解析并标准化返回，顶层提供常用字段，同时保留上游原始 `data` 数据。

```json
{
  "success": true,
  "video_url": "https://v16m.tiktokcdn-us.com/.../video.mp4",
  "cover_url": "https://p16-common-sign.tiktokcdn-us.com/.../cover.jpg",
  "desc": "视频描述",
  "author": "@boise_brooke",
  "data": {
    "aweme_detail": {
      "aweme_id": "7665796670227008781",
      "desc": "视频描述",
      "author": {
        "unique_id": "@boise_brooke",
        "nickname": "Boise❤️Brooke"
      },
      "video": {
        "cover": {
          "url": "https://p16-common-sign.tiktokcdn-us.com/.../cover.jpg"
        },
        "play_addr": {
          "url_list": ["https://v16m.tiktokcdn-us.com/.../video.mp4"]
        },
        "download_addr": {
          "url_list": ["https://v16m.tiktokcdn-us.com/.../video.mp4"]
        },
        "download_no_watermark_addr": {
          "url_list": ["https://v16m.tiktokcdn-us.com/.../video.mp4"]
        }
      },
      "statistics": {
        "digg_count": 123,
        "comment_count": 45,
        "share_count": 6,
        "play_count": 7890
      }
    }
  }
}
```

## 常见返回字段

| 字段 | 说明 |
|------|------|
| `video_url` | **无水印视频下载地址**（推荐直接使用）。优先从 `data.aweme_detail.video.download_no_watermark_addr.url_list[0]` 提取，不存在时回退 `download_addr` / `play_addr`。 |
| `cover_url` | 视频封面图地址。 |
| `desc` | 视频描述/文案。 |
| `author` | 作者用户名（`unique_id`），不存在时回退 `nickname`。 |
| `data` | 上游 TikHub 返回的原始 `data` 层数据，便于访问完整字段。 |
| `data.aweme_detail.video.play_addr.url_list` | 播放地址（可能带水印）。 |
| `data.aweme_detail.video.download_addr.url_list` | 下载地址。 |
| `data.aweme_detail.video.download_no_watermark_addr.url_list` | 无水印下载地址。 |
| `data.aweme_detail.statistics.digg_count` | 点赞数。 |
| `data.aweme_detail.statistics.comment_count` | 评论数。 |
| `data.aweme_detail.statistics.share_count` | 分享数。 |
| `data.aweme_detail.statistics.play_count` | 播放数。 |
| `data.aweme_detail.author.unique_id` | 作者用户名。 |
| `data.aweme_detail.author.nickname` | 作者昵称。 |

## 计费说明

- 后台价格配置项：`video-by-share-url`
- 名称：`通过分享链接获取视频`
- 默认价格：0.01 USD / 次 = 1 积分（按当前汇率换算）
- 管理员免费；普通 / VIP / SVIP 用户按等级价格计费
- 无免费额度时直接按价扣费

## 错误响应

### 400 - 缺少参数

```json
{
  "success": false,
  "message": "share_url 不能为空"
}
```

### 503 - 服务未启用

```json
{
  "success": false,
  "message": "TikHub 接口未启用"
}
```

或

```json
{
  "success": false,
  "message": "TikHub API Key 未配置"
}
```

### 502 - 上游失败

```json
{
  "success": false,
  "message": "tikhub api returned status ..."
}
```

## 对接代码示例 (JavaScript)

```javascript
const response = await fetch(
  'https://heharse.cloud/api/public/tikhub/tiktok/video-by-share-url?share_url=' +
    encodeURIComponent('https://www.tiktok.com/@boise_brooke/video/7665796670227008781'),
  {
    method: 'GET',
    headers: {
      'Authorization': 'Bearer your_newapi_token',
      'Accept': 'application/json'
    }
  }
);
const result = await response.json();
if (result.success) {
  console.log('无水印视频地址:', result.video_url);
  console.log('封面:', result.cover_url);
  console.log('描述:', result.desc);
  console.log('作者:', result.author);
} else {
  console.error('获取失败:', result.message);
}
```

## 注意事项

1. 当前 Base URL 已切换到 `https://heharse.cloud`（旧 `https://api.tikhub.io` 已弃用）。
2. 顶层 `video_url` 字段需要 new-api 更新到最新版本后才会返回；老版本仍可直接透传上游 JSON，需自行从 `data.aweme_detail.video.download_no_watermark_addr.url_list[0]` 提取。
3. 视频下载地址可能存在时效性或访问限制，建议获取后尽快使用或后端转发下载。
