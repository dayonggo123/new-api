#!/bin/bash
# 批量为 articles 生成 FAQ
# 用法：
#   1. 先登录管理后台，从浏览器 Cookie 或 LocalStorage 复制 JWT token
#   2. 执行：bash batch_generate_article_faq.sh "your_jwt_token"

set -e

TOKEN="$1"
if [ -z "$TOKEN" ]; then
  echo "用法: bash batch_generate_article_faq.sh <jwt_token>"
  echo "示例: bash batch_generate_article_faq.sh \"eyJhbGciOiJ...\""
  exit 1
fi

API_BASE="${API_BASE:-http://localhost:3000}"
BATCH_SIZE=20

# MySQL 连接参数（按需修改）
MYSQL_HOST="${MYSQL_HOST:-172.17.0.1}"
MYSQL_PORT="${MYSQL_PORT:-13306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PASS="${MYSQL_PASS:-}"
MYSQL_DB="${MYSQL_DB:-newapi}"

if [ -z "$MYSQL_PASS" ]; then
  echo -n "请输入 MySQL root 密码: "
  read -s MYSQL_PASS
  echo
fi

PASS_ARG=""
if [ -n "$MYSQL_PASS" ]; then
  PASS_ARG="-p$MYSQL_PASS"
fi

echo "查询缺少 FAQ 的 articles..."
IDS=$(mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" $PASS_ARG "$MYSQL_DB" -N -e \
  "SELECT id FROM articles WHERE faq IS NULL OR faq = '' OR faq = '[]' ORDER BY id DESC;")

TOTAL=$(echo "$IDS" | grep -c '^[0-9]\+$' || true)
if [ "$TOTAL" -eq 0 ]; then
  echo "没有需要生成 FAQ 的 articles"
  exit 0
fi

echo "共找到 $TOTAL 条缺少 FAQ 的 articles，每批 $BATCH_SIZE 条触发..."

BATCH=""
COUNT=0
BATCH_NUM=0

call_batch_api() {
  local ids_json="$1"
  local ids_arr=$(echo "$ids_json" | sed 's/,*$//')
  echo "  触发第 $BATCH_NUM 批: [$ids_arr]"
  RES=$(curl -s -X POST "$API_BASE/api/admin/articles/auto-faq/batch" \
    -H "Authorization: $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"ids\":[$ids_arr]}")
  echo "    响应: $RES"
}

for ID in $IDS; do
  BATCH="${BATCH}${ID},"
  COUNT=$((COUNT + 1))

  if [ "$COUNT" -eq "$BATCH_SIZE" ]; then
    BATCH_NUM=$((BATCH_NUM + 1))
    call_batch_api "$BATCH"
    BATCH=""
    COUNT=0
    # 每批间隔 2 秒，避免瞬间压垮 API
    sleep 2
  fi
done

# 处理剩余不足一批的
if [ "$COUNT" -gt 0 ]; then
  BATCH_NUM=$((BATCH_NUM + 1))
  call_batch_api "$BATCH"
fi

echo "全部 $TOTAL 条 articles 的 FAQ 生成任务已提交"
