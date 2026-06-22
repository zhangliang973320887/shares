#!/usr/bin/env bash
# 股价估值分析工具 启动脚本
set -e

cd "$(dirname "$0")"

PORT=5000
PYTHON=${PYTHON:-python3}

echo "==> 检查依赖..."
if ! $PYTHON -c "import flask, akshare, requests" 2>/dev/null; then
  echo "    缺失依赖, 安装中..."
  $PYTHON -m pip install --break-system-packages -q -r requirements.txt
fi

echo "==> 清理 $PORT 端口..."
EXISTING=$(lsof -ti:$PORT 2>/dev/null || true)
if [ -n "$EXISTING" ]; then
  echo "    旧进程 PID=$EXISTING, kill"
  kill -9 $EXISTING 2>/dev/null || true
  sleep 1
fi

echo "==> 启动 Flask..."
$PYTHON app.py &
PID=$!
echo "    PID=$PID"

echo "==> 等待服务就绪..."
for i in $(seq 1 30); do
  if curl -s -o /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null; then
    echo "    就绪!"
    break
  fi
  sleep 0.5
done

URL="http://127.0.0.1:$PORT"
echo ""
echo "================================================"
echo " 服务已启动: $URL"
echo " 停服: kill $PID  或  lsof -ti:$PORT | xargs kill"
echo "================================================"

# 自动开浏览器
case "$(uname -s)" in
  Darwin*) open "$URL" ;;
  Linux*)  xdg-open "$URL" 2>/dev/null || true ;;
  MINGW*|CYGWIN*) start "$URL" ;;
esac

# 保持前台, Ctrl+C 退出
trap "echo ''; echo '==> 停服'; kill $PID 2>/dev/null; exit 0" INT TERM
wait $PID
