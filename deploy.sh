#!/bin/bash
# 部署脚本：拉取最新代码并重新构建部署
# 用法：在服务器项目目录下执行：bash deploy.sh

set -e

# 自动检测项目路径
if [ -d "/home/lighthouse/new-api" ]; then
    PROJECT_DIR="/home/lighthouse/new-api"
elif [ -d "/root/new-api" ]; then
    PROJECT_DIR="/root/new-api"
else
    echo "❌ 未找到项目目录，请手动指定："
    echo "   /home/lighthouse/new-api  或  /root/new-api"
    exit 1
fi

cd "$PROJECT_DIR"
echo "📁 项目目录: $PROJECT_DIR"

echo ""
echo "=== 1. 备份关键文件 ==="
cp router/api-router.go router/api-router.go.bak.$(date +%s) 2>/dev/null || true
cp controller/subscription.go controller/subscription.go.bak.$(date +%s) 2>/dev/null || true

echo ""
echo "=== 2. 拉取最新代码 ==="
git fetch origin stable
git reset --hard origin/stable

echo ""
echo "=== 3. 验证 Go 编译 ==="
if ! go build -o /tmp/new-api-test ./... 2>/dev/null; then
    echo "⚠️ Go 编译检查失败，继续尝试 Docker 构建..."
else
    echo "✅ Go 编译检查通过"
    rm -f /tmp/new-api-test
fi

echo ""
echo "=== 4. 停止旧容器 ==="
docker compose down

echo ""
echo "=== 5. 重新构建并启动 ==="
docker compose up -d --build

echo ""
echo "=== 6. 等待服务启动 ==="
sleep 5

echo ""
echo "=== 7. 健康检查 ==="
for i in {1..10}; do
    if curl -sf http://localhost:3000/api/status | grep -q '"success":\s*true'; then
        echo "✅ 服务启动成功"
        break
    fi
    if [ "$i" -eq 10 ]; then
        echo "❌ 服务启动超时，请检查日志：docker compose logs -f new-api"
        exit 1
    fi
    echo "  等待服务启动... ($i/10)"
    sleep 3
done

echo ""
echo "========================================"
echo "✅ 部署完成"
echo "========================================"
echo ""
echo "验证命令:"
echo "  curl http://localhost:3000/api/status"
echo "  curl -H 'Authorization: Bearer <token>' http://localhost:3000/api/subscription/plans"
echo "  curl -H 'Authorization: Bearer <token>' http://localhost:3000/api/subscription/group-discount"
echo ""
echo "查看日志:"
echo "  docker compose logs -f new-api"
echo ""
