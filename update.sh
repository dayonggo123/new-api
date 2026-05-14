#!/bin/bash
# ============================================
# New-API 服务器更新脚本
# 适用于: 宝塔 + 轻量应用服务器 / 任意 Linux
# ============================================

set -e

# 项目路径（根据你的实际路径修改）
PROJECT_DIR="/www/wwwroot/new-api"

echo "🚀 开始更新 New-API..."
cd "$PROJECT_DIR"

# 1. 拉取最新代码
echo "📥 拉取代码..."
git pull origin stable

# 2. 编译前端
echo "📦 编译前端..."
cd web
npm install
DISABLE_ESLINT_PLUGIN='true' npm run build
cd ..

# 3. 编译后端
echo "🔨 编译后端..."
go mod download
CGO_ENABLED=0 go build -ldflags "-s -w" -o new-api

# 4. 重启服务
echo "🔄 重启服务..."
if systemctl is-active --quiet new-api; then
    systemctl restart new-api
    echo "✅ systemd 服务已重启"
else
    echo "⚠️ 请手动重启你的服务（如 Supervisor / 终端）"
fi

echo "✅ 更新完成"
