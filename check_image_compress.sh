#!/bin/bash
# 检查 new-api 图片压缩日志
# 用法: bash check_image_compress.sh [日志文件路径]
# 如果不传参数，默认检查 logs/ 目录下最新的 .log 文件

LOG_FILE="${1:-}"

if [ -z "$LOG_FILE" ]; then
    # 自动查找最新的日志文件
    if [ -d "logs" ]; then
        LOG_FILE=$(ls -t logs/*.log 2>/dev/null | head -1)
    fi
    if [ -z "$LOG_FILE" ]; then
        echo "未找到日志文件。请指定日志路径，例如:"
        echo "  bash check_image_compress.sh logs/oneapi-20260522.log"
        echo "或者查看 Docker 日志:"
        echo "  docker logs new-api 2>&1 | grep image_compress"
        exit 1
    fi
fi

echo "=========================================="
echo "检查日志文件: $LOG_FILE"
echo "=========================================="
echo ""

# 1. 压缩成功记录
echo "--- 压缩成功记录 ---"
grep "\[image_compress\] compressed:" "$LOG_FILE" 2>/dev/null | tail -20
SUCCESS_COUNT=$(grep -c "\[image_compress\] compressed:" "$LOG_FILE" 2>/dev/null || echo 0)
echo ""
echo "总计成功压缩: $SUCCESS_COUNT 次"
echo ""

# 2. 无需压缩记录
echo "--- 无需压缩记录（<=10MB） ---"
grep "\[image_compress\] no compression needed:" "$LOG_FILE" 2>/dev/null | tail -10
NOOP_COUNT=$(grep -c "\[image_compress\] no compression needed:" "$LOG_FILE" 2>/dev/null || echo 0)
echo ""
echo "总计无需压缩: $NOOP_COUNT 次"
echo ""

# 3. 压缩失败记录
echo "--- 压缩失败记录 ---"
grep "\[image_compress\] compress failed" "$LOG_FILE" 2>/dev/null | tail -10
FAIL_COUNT=$(grep -c "\[image_compress\] compress failed" "$LOG_FILE" 2>/dev/null || echo 0)
echo ""
echo "总计压缩失败: $FAIL_COUNT 次"
echo ""

# 4. 汇总统计
echo "=========================================="
echo "汇总"
echo "=========================================="
echo "成功压缩: $SUCCESS_COUNT"
echo "无需压缩: $NOOP_COUNT"
echo "压缩失败: $FAIL_COUNT"
echo ""

# 5. 如果没有任何记录，给出提示
TOTAL=$((SUCCESS_COUNT + NOOP_COUNT + FAIL_COUNT))
if [ "$TOTAL" -eq 0 ]; then
    echo "⚠️  日志中未找到任何 [image_compress] 记录。"
    echo ""
    echo "可能原因:"
    echo "  1. 应用未重启，新代码未生效"
    echo "  2. 请求未经过 zhangyuge 或 bogei 渠道"
    echo "  3. 请求中的图片不是 base64 data URI（可能是 URL 或文件上传）"
    echo "  4. 日志文件不对，请检查其他日志文件"
    echo ""
    echo "建议:"
    echo "  - 先重启应用使新代码生效"
    echo "  - 发送一个带 base64 图片的请求"
    echo "  - 然后重新运行此脚本"
fi
