#!/bin/bash
# 服务器磁盘清理 + 防止再满配置脚本
# 适用：宝塔面板 + Docker MySQL + Nginx

set -e

echo "=== 1. 清理 MySQL binlog（保留最近 1 天）==="
docker exec -i mysql_wrmh-mysql_WrMh-1 mysql -u root -p"${MYSQL_ROOT_PASSWORD:-}" -e "PURGE BINARY LOGS BEFORE DATE(NOW() - INTERVAL 1 DAY);" || echo "MySQL 清理失败，请检查 root 密码"

echo "=== 2. 清理 Docker 容器日志 ==="
for log in /var/lib/docker/containers/*/*-json.log; do
    if [ -f "$log" ]; then
        > "$log"
        echo "已清空: $log"
    fi
done

echo "=== 3. 清理 Nginx 访问日志 ==="
LOG_FILE="/www/wwwlogs/heharse.cloud.log"
if [ -f "$LOG_FILE" ]; then
    mv "$LOG_FILE" "$LOG_FILE.bak.$(date +%F)"
    touch "$LOG_FILE"
    chown www:www "$LOG_FILE"
    echo "已归档并清空: $LOG_FILE"
fi

echo "=== 4. 限制 Docker 日志大小 ==="
if [ ! -f /etc/docker/daemon.json ]; then
    cat > /etc/docker/daemon.json <<'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "50m",
    "max-file": "3"
  }
}
EOF
    echo "Docker 日志限制已配置，需重启 Docker 生效"
else
    echo "/etc/docker/daemon.json 已存在，请手动合并日志限制配置"
fi

echo "=== 5. 当前磁盘使用情况 ==="
df -h /

echo "=== 完成 ==="
