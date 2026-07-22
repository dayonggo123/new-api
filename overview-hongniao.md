# HongniaoAI（红鸟 AI）渠道接入完成（OpenAPI 版）

## 完成内容

按真实 OpenAPI 文档修正了 HongniaoAI 渠道：

- Base URL 更新为 `https://open.hongniaoai.com/`
- 鉴权方式改为 `Authorization: Bearer {key}`
- 同时支持视频生成与图片生成任务

## 上游接口

- 视频提交：`POST /v1/videos`
- 视频查询：`GET /v1/videos/{id}`
- 图片提交：`POST /v1/images`
- 图片查询：`GET /v1/images/{id}`
- 状态流：`queued` → `processing` → `completed` / `failed`

## 修改文件

1. `constant/channel.go` — Base URL 更新为 `https://open.hongniaoai.com/`
2. `relay/channel/task/hongniao/adaptor.go` — 重写适配器：Bearer 鉴权、同时支持图片/视频、按请求路径自动判断任务类型
3. `common/endpoint_type.go` — Hongniao 支持 `EndpointTypeOpenAIVideo` 与 `EndpointTypeImageGeneration`
4. `relay/image_handler.go` — Hongniao 图片请求走 task 异步路径
5. `controller/channel-test.go` — 已支持 Hongniao 视频/图片渠道测试路由
6. `setting/ratio_setting/model_ratio.go` — 保留 `keling-3` / `sdquan-2` 默认按次价格 `0.3`（用户可在后台覆盖）
7. `web/src/constants/channel.constants.js` — 前端渠道下拉已存在 `HongniaoAI`

## 验证

- `go build ./...` 编译通过
- 本地 `go test` 因测试二进制运行环境架构问题无法执行，不影响代码编译

## 后台配置建议

- 渠道类型：HongniaoAI
- Base URL：`https://open.hongniaoai.com/`
- 模型列表：按上游实际模型自行填写
- 模型价格：在「模型价格」页面按次配置

## 测试示例

视频：
```bash
curl -X POST https://heharse.cloud/v1/videos/generations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"keling-3","prompt":"a beautiful sunset over ocean","aspectRatio":"9:16","seconds":"10"}'
```

图片：
```bash
curl -X POST https://heharse.cloud/v1/images/generations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image2","prompt":"a beautiful sunset over ocean","aspect_ratio":"1:1"}'
```
