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

### seedance 参考图支持
- **问题**: seedance 系列模型上传参考图后 `reference_item` 为空
- **原因**: `ref_images` 字段被错误地重命名为 `files`，upstream 无法识别
- **修复**: 保持原始字段名 `ref_images`/`ref_videos`/`ref_audios` 原样转发

### seedance mode/duration 字段
- **修复**: 新增 `mode` 和 `duration` 字段读取与转发

### reference_item 解析
- **修复**: 在 `historyItem` 结构体中新增 `ReferenceItem` 字段，并传递到 `TaskInfo`
