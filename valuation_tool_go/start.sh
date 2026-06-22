#!/usr/bin/env bash
# 股价估值分析工具 (Go) 启动脚本
set -e

cd "$(dirname "$0")"

# 强制 Go module 模式 (老 Go 默认 GOPATH 模式会找不到 internal 包)
export GO111MODULE=on
export GOPROXY=${GOPROXY:-https://goproxy.cn,direct}

PORT=${PORT:-5000}
HOST=${HOST:-127.0.0.1}
BIN=${BIN:-./valuation}

# 判断是否需要重新编译: 缺二进制 / .go 比二进制新
need_build=0
if [ ! -x "$BIN" ]; then
  need_build=1
else
  if find . -name "*.go" -newer "$BIN" 2>/dev/null | grep -q .; then
    need_build=1
  fi
fi

if [ "$need_build" = "1" ]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "❌ 缺二进制且未安装 go"
    echo "   方案1: 本机交叉编译后传到服务器"
    echo "          GOOS=linux GOARCH=amd64 go build -o valuation ."
    echo "          scp valuation web data root@server:/opt/valuation/"
    echo "   方案2: 服务器装 Go: https://go.dev/dl/"
    exit 1
  fi
  echo "==> go mod tidy..."
  go mod tidy
  echo "==> 编译..."
  go build -o valuation .
else
  echo "==> 复用已有二进制 $BIN"
fi

# 清端口 (优先 lsof, 退化到 fuser)
echo "==> 清理 $PORT 端口..."
EXISTING=""
if command -v lsof >/dev/null 2>&1; then
  EXISTING=$(lsof -ti:$PORT 2>/dev/null || true)
elif command -v fuser >/dev/null 2>&1; then
  EXISTING=$(fuser $PORT/tcp 2>/dev/null || true)
elif command -v ss >/dev/null 2>&1; then
  EXISTING=$(ss -ltnp "sport = :$PORT" 2>/dev/null | grep -oP 'pid=\K\d+' || true)
fi
if [ -n "$EXISTING" ]; then
  echo "    旧 PID=$EXISTING, kill"
  kill -9 $EXISTING 2>/dev/null || true
  sleep 1
fi

if [ "$HOST" = "127.0.0.1" ]; then
  echo "==> 只监听本机. 公网访问请用: HOST=0.0.0.0 ./start.sh"
fi

echo "==> 启动: http://$HOST:$PORT"
echo "    Ctrl+C 退出"
trap 'echo ""; echo "==> 停服"; exit 0' INT TERM
PORT=$PORT HOST=$HOST exec "$BIN"
