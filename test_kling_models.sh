#!/bin/bash
# 测试 Kling 模型映射是否生效
# 用法: bash test_kling_models.sh <API_KEY>

API_KEY="${1:-YOUR_API_KEY}"
BASE_URL="http://localhost:3000"

models=(
  "kling"
  "kling-video-3-0"
  "kling-video-2-6"
  "kling-video-2-5"
  "kling-video-2-1-5s"
  "kling-video-2-1-10s"
  "kling-video-1-6-10s"
  "kling-video-1-6-5s"
  "kling-video-o1"
  "kling-video-motion-3"
  "kling-video-motion"
  "kling-video-3-0-edit"
  "kling-video-o1-edit"
  "kling-video-lipsync"
)

echo "🧪 测试 Kling 模型映射 ($BASE_URL)..."
echo ""

for model in "${models[@]}"; do
  echo -n "[$model] "
  resp=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "$BASE_URL/v1/video/generations" \
    -H "Authorization: Bearer $API_KEY" \
    -F "model=$model" \
    -F "prompt=test" 2>/dev/null)
  
  if [ "$resp" = "200" ] || [ "$resp" = "201" ]; then
    echo "✅ OK ($resp)"
  elif [ "$resp" = "403" ] || [ "$resp" = "401" ]; then
    echo "⚠️ 鉴权失败 ($resp) — 模型已识别，检查 API Key"
  elif [ "$resp" = "503" ]; then
    echo "⚠️ 无可用渠道 ($resp) — 模型已识别，检查渠道配置"
  else
    echo "❌ 失败 ($resp)"
  fi
done

echo ""
echo "测试完成。200/201/403/503 都表示模型映射已生效（只是后续处理阶段不同）。"
