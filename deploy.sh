#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "[1/4] Pulling latest code..."
git pull

echo "[2/4] Building and deploying..."
docker-compose down
docker-compose up -d --build

echo "[3/4] Cleaning up old images..."
docker image prune -f

echo "[4/4] Waiting for service..."
sleep 5
docker ps | grep new-api

echo ""
echo "Deploy done at $(date)"
