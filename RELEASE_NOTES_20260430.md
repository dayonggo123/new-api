# 发布总结：New-API v0.13.1+ 异步图像生成 & MySQL 迁移

**发布日期**：2026-04-30  
**Git Commit**：`238cde77` on `stable`

---

## 一、新增功能

### 1. 异步图像生成（解决 504 超时）
- **后端**：
  - `POST /v1/images/generations?async=true` — 提交异步任务，立即返回 `task_id`
  - `GET /v1/images/tasks/:task_id` — 轮询查询任务状态，返回结果中的图片 URL 自动替换为持久化代理地址
  - 新增 `service/async_image.go` — 内存任务映射（24h TTL，带过期清理）
  - 新增 `controller/async_image.go` — 轮询 handler + URL rewrite
  - `relay/image_handler.go` — 拦截上游 async 响应，支持 `task_id` 和 `id` 两种字段名

- **前端构建修复**：
  - 修复 `web/src/constants/common.constant.js` 缺失 `TASK_ACTION_VIDEO_GENERATE` 导出的问题

### 2. 持久化图片代理
- `GET /image-proxy/:id` — 懒加载上游图片并本地缓存
- 解决上游临时图片 URL 过期问题
- `/v1/images/generations` 同步响应中的 `url` 自动替换为代理地址

### 3. 文件上传与清理
- `POST /uapi/v1/upload_videos` — 视频上传端点
- `POST /uapi/v1/upload_images` / `/uapi/v1/upload_images/json` — 图片上传
- `service/upload_cleanup.go` — 自动清理 3 天前的上传文件

---

## 二、基础设施变更

### 数据库切换：PostgreSQL -> MySQL
- **原因**：Docker 容器内 PostgreSQL 为空，用户历史数据在宝塔 MySQL 中
- **操作**：
  - 从宝塔备份恢复 `newapi` 数据库（`newapi_2026-04-30_10-50-26_mysql_data_XEgbJ.sql.zip`）
  - 修改 `docker-compose.yml`：`SQL_DSN` 指向宝塔 MySQL（`10.5.0.4:13306`）
  - `docker-compose.yml` 改用 `build: .`（本地源码构建），替代 `image: calciumion/new-api:latest`

---

## 三、Bug 修复

| 问题 | 修复 |
|------|------|
| nano-banana / imagen-4 `ref_images` 不生效 | 扩展 `needsFilesField` 映射到上游 `files` 字段 |
| 前端构建失败 `TASK_ACTION_VIDEO_GENERATE is not exported` | 在 `common.constant.js` 中补全导出 |
| 内存泄漏（async 任务 map） | `RegisterAsyncImageTask` 时触发 opportunistic cleanup |

---

## 四、修改文件清单

```
新增：
  service/async_image.go
  service/async_image_test.go
  controller/async_image.go

修改：
  relay/image_handler.go          (+ async 分支)
  router/relay-router.go          (+ /v1/images/tasks/:task_id 路由)
  web/src/constants/common.constant.js  (+ TASK_ACTION_VIDEO_GENERATE)
  docker-compose.yml              (MySQL DSN + build: .)
  API_DOCUMENTATION.md            (异步调用文档)
```

---

## 五、验证状态

| 项目 | 状态 |
|------|------|
| 代码编译 | 通过 |
| 单元测试 | 3/3 通过 |
| Docker 构建 | 通过 |
| MySQL 数据恢复 | `users` 表 4 条记录，系统已初始化 |
| async 接口 | 提交返回 `task_id`，轮询返回结果 |
| GeminiGen 渠道 | 后端任务轮询正常（nano-banana-2 测试通过） |
| 图片代理 | 临时 URL 替换为持久化代理地址 |

---

## 六、后续注意事项

1. **async 任务计费**：当前 async 模式提交时预扣费，查询时不额外计费。若任务最终失败，预扣费用不退还。后续可在轮询接口补充失败退费逻辑。
2. **Docker 镜像**：已改为本地 `build: .`，后续更新需执行 `docker-compose build --no-cache`。
3. **PostgreSQL 容器**：当前 docker-compose 中仍定义了 postgres 服务（未删除），会启动但不使用。如需清理可注释掉整个 postgres 服务块。
