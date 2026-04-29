# Docker 环境更新指南

## 重要提醒

new-api 后端运行在 Docker 容器中（容器名: `new-api`）。

**宿主机上直接修改 `/usr/local/bin/new-api` 或编译二进制文件不会影响容器内运行的服务。**

## 正确的更新流程

```bash
cd /home/lighthouse/new-api

# 1. 拉取最新代码（或手动修改后）
git pull

# 2. 编译新版本
go build -o new-api

# 3. 复制新二进制到容器内
docker cp /home/lighthouse/new-api/new-api 45fdff47b41a:/usr/local/bin/new-api

# 4. 重启容器
docker restart 45fdff47b41a

# 5. 确认状态
docker ps | grep new-api
curl -s http://localhost:3000/api/status | head -c 50
```

## 常见问题排查

### 端口被占用 / 无法启动
```bash
# 查看占用 3000 端口的进程
ss -tlnp | grep 3000

# 如果是 Docker 容器占用了端口，停止并重启容器即可
docker restart 45fdff47b41a
```

### 确认运行的是最新版本
```bash
# 查看容器内二进制文件的修改时间
docker exec new-api ls -la /usr/local/bin/new-api

# 查看容器状态
docker ps | grep new-api
```

### 日志查看
```bash
# 查看容器日志
docker logs new-api

# 查看应用日志（容器内）
docker exec new-api ls -la /app/logs/
```

## 2026-04-29 修复记录

### OpenAI 图片接口懒加载代理
- **新增**: `/image-proxy/:id` 公共端点，解决 OpenAI/DALL-E 等上游返回的临时图片 URL 过期问题
- **机制**: `/v1/images/generations` 返回的 `url` 自动替换为 `https://heharse.cloud/image-proxy/{uuid}.png`；客户端首次访问时后端才从上游拉取并本地缓存
- **优点**: 下游聊天客户端永久可展示生成的图片

### nano-banana / imagen-4 参考图支持
- **问题**: nano-banana-2 / nano-banana-pro / imagen-4 模型上传参考图后未生效
- **原因**: 这些模型上游与 grok 一样，要求参考图使用 `files` 字段，但代码只给 grok 做了映射
- **修复**: 将 `needsFilesField` 扩展到 `nano-banana-*` 和 `imagen-4` 模型，`ref_images` 文件自动映射为上游 `files` 字段

### seedance 参考图支持
- **问题**: seedance 系列模型上传参考图后 `reference_item` 为空
- **原因**: `ref_images` 字段被错误地重命名为 `files`，upstream 无法识别
- **修复**: 保持原始字段名 `ref_images`/`ref_videos`/`ref_audios` 原样转发

### seedance mode/duration 字段
- **修复**: 新增 `mode` 和 `duration` 字段读取与转发

### reference_item 解析
- **修复**: 在 `historyItem` 结构体中新增 `ReferenceItem` 字段，并传递到 `TaskInfo`

### grok 参考图字段映射
- **问题**: grok 上游只接受 `files`（multipart 文件）、`file_urls`（URL 字符串）、`ref_images`（UUID 字符串）三种互斥方式，下游统一传 `ref_images` 会报 400
- **修复**: grok 模型下，`ref_images` 文件自动映射为上游 `files` 字段；`ref_images` URL 自动映射为 `file_urls`；UUID 字符串保持为 `ref_images`

### GeminiGen 错误解析
- **问题**: 上游错误格式为 `{"detail": {"error_code": "...", "error_message": "..."}}`，代码只支持 `message` 字段
- **修复**: `dto/error.go` 中 `GeminiGenErrorDetail` 同时支持 `message` 和 `error_message` 字段；提交和轮询阶段均提取友好错误信息

### 轮询提取 processing 状态的 URL
- **问题**: 部分平台（grok）在 `status: 1`（processing）时就返回了 `video_url`，但代码只在 `status: 2`（completed）时提取
- **修复**: `ParseTaskResult` 在 processing 状态也提取 `video_url`、`image_url` 和 `reference_item`；轮询循环只要有 URL 就立即保存到 `ResultURL`

### 轮询跳过 progress=100% 的任务
- **问题**: `GetAllUnFinishSyncTasks` 排除了 `progress="100%"` 的任务，导致 grok 任务在上游返回 `status_percentage=100` 但 `status=1` 时被轮询跳过，永远无法到达终态
- **修复**: 移除 `progress != "100%"` 过滤条件，轮询只根据 `status` 判断（排除 SUCCESS/FAILURE）
