# New-API 部署检查清单

> 基于 2026-04-30 异步图片生成 + MySQL 迁移发布过程的经验总结。
> 按此清单执行，可避免 90% 的部署踩坑。

---

## 一、部署前检查

### 1.1 代码同步确认

- [ ] 确认所有修改已提交到 Git
- [ ] 确认服务器已拉取最新代码（`git pull` 或手动同步）
- [ ] **关键**：在服务器上直接 `grep` 验证新增导出是否到位

```bash
cd /home/lighthouse/new-api  # 确认实际路径
grep -n "TASK_ACTION_IMAGE_GENERATE\|TASK_ACTION_VIDEO_GENERATE" \
  web/src/constants/common.constant.js
```

**教训**：本地修改了 `common.constant.js`，但服务器未同步，导致 Docker 构建两次失败。

### 1.2 前端构建预检

如果服务器**没有**安装 node/npm/bun，前端构建只能在 Docker 内进行。部署前建议先验证前端能否编译通过：

```bash
cd web
# 方式一：用 Docker 临时运行 bun（推荐，与 Dockerfile 环境一致）
docker run --rm -v "$(pwd):/web" -w /web oven/bun:alpine \
  sh -c "bun install && DISABLE_ESLINT_PLUGIN='true' bun run build 2>&1" | tail -30
```

**常见构建错误**：
| 错误 | 原因 | 解决 |
|------|------|------|
| `X is not exported by common.constant.js` | 新增常量未在 `web/src/constants/common.constant.js` 导出 | 补全导出 |
| `Use of eval in lottie.js` | 警告，非错误 | 忽略 |
| `Build failed in Xs` + 无具体错误 | 内存不足 | 换 `oven/bun:alpine` 镜像或增加服务器内存 |

### 1.3 确认 Docker Compose 路径

```bash
find / -name "docker-compose.yml" -not -path "*/containerd/*" 2>/dev/null
```

**教训**：实际路径是 `/home/lighthouse/new-api/`，而非文档中的 `/home/ubuntu/new-api/`。

---

## 二、Docker 构建

### 2.1 修改 Dockerfile（如前端构建卡住）

如果之前构建在 `rendering chunks...` 阶段卡住，将前端构建镜像从 `oven/bun:1` 改为 `oven/bun:alpine`：

```bash
sed -i 's|FROM oven/bun:1@sha256:.* AS builder|FROM oven/bun:alpine AS builder|' Dockerfile
```

### 2.2 构建与启动

```bash
cd /home/lighthouse/new-api  # 替换为实际路径
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

**关键**：
- 必须加 `--no-cache`，否则 Go 编译缓存可能导致新代码不生效
- 构建失败时**不要**执行 `docker-compose up -d`，否则会启动旧镜像

### 2.3 构建失败处理

如果构建失败：
1. `Ctrl+C` 终止
2. 查看失败原因（通常是前端构建错误）
3. 修复源码后重新执行 `docker-compose build --no-cache`
4. **不要跳过修复直接 up**

---

## 三、部署后验证

### 3.1 服务启动检查

```bash
docker-compose ps
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/status
```

### 3.2 异步图片接口验证（必做）

| 步骤 | 命令 | 预期结果 |
|------|------|----------|
| 同步图片 URL 代理 | `POST /v1/images/generations` | `url` 字段为 `/image-proxy/xxx.png` |
| 异步提交 | `POST /v1/images/generations?async=true` | 返回 `{"task_id":"..."}` |
| 轮询不存在的任务 | `GET /v1/images/tasks/test123` | 返回 `404` |
| 轮询真实任务 | `GET /v1/images/tasks/{task_id}` | 返回 `code:success` + 代理 URL |
| 图片代理加载 | `GET /image-proxy/{id}.png` | 返回 `200` + 图片数据 |

**完整验证脚本**：

```bash
API_KEY="sk-your-key"
BASE="http://localhost:3000"

echo "1. Sync generation..."
SYNC=$(curl -s -X POST "$BASE/v1/images/generations" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","prompt":"test","size":"1024x1024","n":1}')
echo "$SYNC" | grep -q "image-proxy" && echo "✅ URL rewritten" || echo "❌ URL not rewritten"

echo "2. Async submit..."
ASYNC=$(curl -s -X POST "$BASE/v1/images/generations?async=true" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","prompt":"test","size":"1024x1024","n":1}')
TASK_ID=$(echo "$ASYNC" | grep -oP '"task_id":"\K[^"]+')
echo "Task: $TASK_ID"

echo "3. Poll task..."
sleep 5
POLL=$(curl -s "$BASE/v1/images/tasks/$TASK_ID" \
  -H "Authorization: Bearer $API_KEY")
echo "$POLL" | grep -q "SUCCESS\|completed" && echo "✅ Task completed" || echo "⏳ Still processing"
```

---

## 四、常见问题速查

### Q1: 为什么 `docker-compose build` 后新代码没生效？

A: 99% 是因为构建失败后用旧镜像启动了。检查：
```bash
docker images | grep new-api
docker inspect new-api | grep "Created"
```
如果 `Created` 时间是上次成功构建的时间，说明最新构建失败了。

### Q2: 前端构建卡在 `rendering chunks...` 怎么办？

A: 
1. 检查服务器内存：`free -h`（可用内存 < 1G 容易卡住）
2. 换 `oven/bun:alpine` 镜像
3. 或在外部构建好 `dist` 后让 Dockerfile 直接 COPY

### Q3: 轮询接口返回前端 HTML 而不是 JSON？

A: 说明 `/v1/images/tasks/:task_id` 路由未注册，容器运行的是旧代码。必须重新构建镜像。

### Q4: 同步接口返回的 URL 是上游原始地址？

A: 同样说明镜像未更新。`rewriteImageResponseWithProxyURLs` 只在新代码中存在。

### Q5: `docker-compose ps` 提示 `no configuration file provided`？

A: 当前目录不对。用 `find / -name docker-compose.yml` 找到实际路径。

---

## 五、数据库切换备忘（MySQL/PostgreSQL/SQLite）

### 5.1 切换步骤

1. 备份原数据库
2. 修改 `docker-compose.yml` 中的 `SQL_DSN`
3. 确保 DSN 格式正确：`user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True`
4. 如需本地源码构建，确认 `docker-compose.yml` 中用的是 `build: .` 而非 `image: xxx`

### 5.2 DSN 模板

| 数据库 | DSN 格式 |
|--------|----------|
| MySQL | `root:password@tcp(10.5.0.4:13306)/newapi?charset=utf8mb4&parseTime=True` |
| PostgreSQL | `host=localhost user=postgres password=xxx dbname=newapi sslmode=disable` |
| SQLite | `newapi.db` |

---

## 六、文件清单（本次发布涉及）

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `service/async_image.go` | 新增 | 异步任务内存映射 + 轮询上游 |
| `service/async_image_test.go` | 新增 | 单元测试 |
| `controller/async_image.go` | 新增 | 轮询 handler + URL rewrite |
| `service/image_proxy.go` | 新增 | 图片代理注册 + 懒加载缓存 |
| `controller/image_proxy.go` | 新增 | 图片代理 HTTP handler |
| `relay/image_handler.go` | 修改 | 添加 async 分支 + URL 重写 |
| `router/relay-router.go` | 修改 | 添加 `/v1/images/tasks/:task_id` 路由 |
| `web/src/constants/common.constant.js` | 修改 | 补全 `TASK_ACTION_VIDEO_GENERATE` + `TASK_ACTION_IMAGE_GENERATE` |
| `docker-compose.yml` | 修改 | MySQL DSN + `build: .` |
| `Dockerfile` | 修改 | `oven/bun:1` → `oven/bun:alpine` |

---

*最后更新：2026-04-30*
