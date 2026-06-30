# EchoTik 视频榜单接口接入文档

## 功能概述

将 EchoTik `/api/v3/echotik/video/ranklist` 接口接入 new-api，下游应用只需调用 new-api 的公开接口即可获取 EchoTik 视频榜单数据。

## 已修改/新增文件

| 文件 | 说明 |
|------|------|
| `setting/operation_setting/echotik_setting.go` | EchoTik 配置模块 |
| `controller/echotik.go` | 接口代理逻辑 |
| `router/api-router.go` | 路由注册 |
| `web/src/pages/Setting/Operation/SettingsEchotik.jsx` | 前端配置页面 |
| `web/src/components/settings/OperationSetting.jsx` | 引入配置页面 |

## 管理后台配置

1. 登录 new-api 管理后台
2. 进入 **系统设置 → 运营设置 → EchoTik 设置**
3. 填写：
   - 启用 EchoTik 接口代理
   - EchoTik API 基础地址：`https://open.echotik.live`
   - EchoTik 用户名：`250828757562492145`
   - EchoTik 密码：`b9161a6908724992ac70ba0714777c8a`
4. 点击保存

## 下游应用调用方式

### 接口地址

```
GET https://你的new-api域名/api/public/echotik/video/ranklist
```

### 鉴权

在请求头中传入 new-api 的 API Key：

```
Authorization: Bearer sk-xxxxxx
```

### 参数（原样透传给 EchoTik）

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| date | string | 是 | yyyy-MM-dd 格式 |
| region | string | 是 | 区域代码，如 US |
| video_rank_field | integer | 是 | 1=热门榜(total_views_cnt)，2=带货榜(total_video_sale_cnt) |
| rank_type | integer | 是 | 1=日榜，2=周榜，3=月榜 |
| page_num | integer | 是 | 页码，从 1 开始 |
| page_size | integer | 是 | 每页条数，最大 10 |
| product_category_id | string | 否 | 商品一级类目 ID |
| created_by_ai | string | 否 | true/false，是否 AI 视频 |

### 示例

```bash
curl -X GET 'https://你的new-api域名/api/public/echotik/video/ranklist?date=2026-06-29&region=US&video_rank_field=1&rank_type=1&page_num=1&page_size=10' \
  --header 'Authorization: Bearer sk-xxxxxx'
```

### 响应

直接透传 EchoTik 的原始 JSON 响应：

```json
{
  "code": 200,
  "message": "success",
  "requestId": "xxx",
  "data": [
    {
      "video_id": "xxx",
      "nick_name": "xxx",
      "total_views_cnt": 1000000,
      ...
    }
  ]
}
```

## 安全说明

- EchoTik 的 username/password 保存在 new-api 数据库中，不会暴露给下游应用
- 下游应用仅需要 new-api 的 API Key
- 管理后台可查看配置状态：`GET /api/admin/echotik/status`
