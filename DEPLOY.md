# New-API 发布部署指南

## 环境信息

| 项目 | 值 |
|---|---|
| 服务器 | 腾讯云 Lighthouse（轻量应用服务器） |
| 域名 | heharse.cloud |
| 部署路径 | `/home/lighthouse/new-api` |
| 部署方式 | Docker |
| 容器名 | `new-api` |
| 端口 | `3000` |

## 方案一：Docker 构建部署（推荐，服务器无 Go/npm）

```bash
cd /home/lighthouse/new-api
git pull origin stable
docker-compose down
docker-compose up -d --build
```

如果 Docker 版本较新（v2+）：
```bash
cd /home/lighthouse/new-api
git pull origin stable
docker compose down
docker compose up -d --build
```

**优点**：不需要服务器安装 Go/Node，构建环境自包含  
**缺点**：构建时间较长，可能触发 OOM

## 方案二：二进制热更新（更快，需本地编译）

### 本地（Windows）交叉编译

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o new-api-linux
```

### 上传到服务器

```bash
scp -i ~/.ssh/lhkp-4hrmuc6n.pem new-api-linux root@heharse.cloud:/home/lighthouse/new-api/
```

### 服务器替换并重启

```bash
cd /home/lighthouse/new-api
docker cp new-api-linux new-api:/usr/local/bin/new-api
docker restart new-api
docker ps | grep new-api
curl -s http://localhost:3000/api/status | head -c 50
```

**优点**：几秒完成更新，不重建镜像  
**缺点**：需要本地有 Go 环境

## 查看容器状态

```bash
# 查看运行中的容器
docker ps | grep new-api

# 查看容器日志
docker logs new-api

# 查看应用日志（容器内）
docker exec new-api ls -la /app/logs/

# 验证服务状态
curl -s http://localhost:3000/api/status | head -c 50
```

## 常见问题

### 端口占用
```bash
ss -tlnp | grep 3000
docker restart new-api
```

### 确认运行的是最新版本
```bash
docker exec new-api ls -la /usr/local/bin/new-api
```
