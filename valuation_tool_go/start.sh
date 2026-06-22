#!/usr/bin/env bash
# Go 版股价估值分析工具
set -e

cd "$(dirname "$0")"

PORT=${PORT:-5000}
HOST=${HOST:-127.0.0.1}

if ! command -v go >/dev/null 2>&1; then
  echo "❌ 未找到 go 命令, 安装: https://go.dev/dl/"
  exit 1
fi

echo "==> go mod tidy..."
go mod tidy

echo "==> 清理 $PORT 端口..."
EXISTING=$(lsof -ti:$PORT 2>/dev/null || true)
[ -n "$EXISTING" ] && kill -9 $EXISTING 2>/dev/null || true

echo "==> 编译..."
go build -o valuation .

echo "==> 启动: http://$HOST:$PORT"
PORT=$PORT HOST=$HOST ./valuation
