#!/bin/bash
LOG=$(ls -t logs/*.log 2>/dev/null | head -1)
if [ -z "$LOG" ]; then echo "未找到日志"; exit 1; fi
echo "日志: $LOG"
echo ""
echo "--- image_compress 相关记录 ---"
grep "image_compress" "$LOG" | tail -30
echo ""
echo "--- 统计 ---"
echo "compressed: $(grep -c 'compressed:' "$LOG" 2>/dev/null || echo 0)"
echo "no compression: $(grep -c 'no compression needed' "$LOG" 2>/dev/null || echo 0)"
echo "failed: $(grep -c 'compress failed' "$LOG" 2>/dev/null || echo 0)"
