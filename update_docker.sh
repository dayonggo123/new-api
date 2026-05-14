#!/bin/bash
# ============================================
# New-API Docker 一键更新脚本
# 适用于: 腾讯云 Lighthouse + Docker 部署
# 路径: /home/lighthouse/new-api
# ============================================

set -e

PROJECT_DIR="/home/lighthouse/new-api"
CONTAINER_NAME="new-api"

cd "$PROJECT_DIR"

echo "📥 拉取最新代码..."
git pull origin stable

echo "🛑 停止旧容器..."
docker-compose down 2>/dev/null || docker compose down 2>/dev/null || true

echo "🔨 构建并启动新容器..."
if command -v docker-compose &> /dev/null; then
    docker-compose up -d --build
else
    docker compose up -d --build
fi

echo "⏳ 等待服务启动..."
sleep 5

echo "📋 容器状态:"
docker ps | grep "$CONTAINER_NAME"

echo "🩺 健康检查:"
curl -s http://localhost:3000/api/status | head -c 100
echo ""

echo "✅ 部署完成"
