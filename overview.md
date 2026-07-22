# Z-api（Seedance 2.0-fast）渠道接入完成

## 状态

已接入并可编译通过。后端适配器此前已存在，本次补齐前端渠道下拉。

## 关键文件

| 文件 | 说明 |
|------|------|
| `constant/channel.go` | `ChannelTypeZAPI = 67`，Base URL `https://api.tmlab.store` |
| `common/api_type.go` | ZAPI → `APITypeOpenAI` |
| `common/endpoint_type.go` | ZAPI → `EndpointTypeOpenAIVideo` |
| `relay/relay_adaptor.go` | 注册 `task/zapi` 任务适配器 |
| `relay/channel/task/zapi/adaptor.go` | Seedance 异步任务提交/查询适配 |
| `controller/relay.go` | 透传 `duration`/`ratio`/`resolution`/`images`/`videos`/`audios`/`first_image`/`last_image` |
| `controller/channel-test.go` | ZAPI 渠道测试路由到 `/v1/videos/generations` |
| `setting/ratio_setting/model_ratio.go` | `seedance-2.0-fast(431)` 默认按次价格 `3.0/7.3` |
| `web/src/constants/channel.constants.js` | 新增 `value: 67, label: 'ZAPI'` |

## 验证

- `go build ./...` 通过。

## 测试示例

```bash
curl -X POST https://heharse.cloud/v1/videos/generations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "seedance-2.0-fast(431)",
    "prompt": "一只可爱的小猫在草地上玩耍，电影感风格，镜头缓慢推进",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

## 注意

- ZAPI 为按次计费模型，需确保模型价格配置启用 `UsePrice` 或 `TASK_PRICE_PATCHES` 包含该模型。
- 实际可用性需要上游 API Key 与额度验证。
