#!/bin/bash
# 检查 new-api 中已配置的模型固定价格（按次计费）
# 用法: bash check_model_price.sh

echo "=========================================="
echo "1. 环境变量 TASK_PRICE_PATCHES"
echo "=========================================="
if [ -n "$TASK_PRICE_PATCHES" ]; then
    echo "$TASK_PRICE_PATCHES"
else
    echo "(未设置)"
fi

echo ""
echo "=========================================="
echo "2. Docker Compose 中的 TASK_PRICE_PATCHES"
echo "=========================================="
if [ -f docker-compose.yml ]; then
    grep -i "TASK_PRICE_PATCHES" docker-compose.yml || echo "(未找到)"
elif [ -f compose.yml ]; then
    grep -i "TASK_PRICE_PATCHES" compose.yml || echo "(未找到)"
else
    echo "(未找到 docker-compose.yml 或 compose.yml)"
fi

echo ""
echo "=========================================="
echo "3. 数据库中的 ModelPrice 配置（按次计费价格）"
echo "=========================================="
DB_FILE="data/one-api.db"
if [ -f "$DB_FILE" ]; then
    echo "数据库: $DB_FILE (SQLite)"
    sqlite3 "$DB_FILE" "SELECT value FROM options WHERE key = 'ModelPrice';" 2>/dev/null | python3 -m json.tool 2>/dev/null || sqlite3 "$DB_FILE" "SELECT value FROM options WHERE key = 'ModelPrice';" 2>/dev/null
    if [ $? -ne 0 ]; then
        echo "(sqlite3 或 python3 不可用，原始输出:)"
        sqlite3 "$DB_FILE" "SELECT value FROM options WHERE key = 'ModelPrice';" 2>/dev/null
    fi
else
    echo "(未找到 SQLite 数据库 $DB_FILE)"
fi

echo ""
echo "=========================================="
echo "4. 代码中硬编码的 defaultModelPrice"
echo "=========================================="
GO_FILE="setting/ratio_setting/model_ratio.go"
if [ -f "$GO_FILE" ]; then
    awk '/var defaultModelPrice = map\[string\]float64\{/,/\}/' "$GO_FILE"
else
    echo "(未找到 $GO_FILE)"
fi

echo ""
echo "=========================================="
echo "5. 代码中硬编码的 defaultModelRatio（按 token 计费）"
echo "=========================================="
if [ -f "$GO_FILE" ]; then
    echo "(文件太大，请直接查看 $GO_FILE 的 defaultModelRatio 部分)"
else
    echo "(未找到 $GO_FILE)"
fi

echo ""
echo "=========================================="
echo "说明"
echo "=========================================="
echo "- 只有同时满足以下两个条件，模型才会按次计费:"
echo "  1. ModelPrice 中有该模型的价格（上面第3点或第4点）"
echo "  2. TASK_PRICE_PATCHES 包含该模型名，或价格>0且UsePrice=true"
echo "- 第3点（数据库）的优先级高于第4点（代码硬编码）"
echo "- 如果模型只在 TASK_PRICE_PATCHES 中但不在 ModelPrice 中，"
echo "  系统会标记 PerCallBilling=true，但价格可能 fallback 到按 token 计费"
