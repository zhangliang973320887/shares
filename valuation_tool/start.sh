#!/usr/bin/env bash
# 股价估值分析工具 启动脚本
set -e

cd "$(dirname "$0")"

PORT=5000
PYTHON=${PYTHON:-python3}

echo "==> 检查 Python 版本..."
PY_VER=$($PYTHON -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
PY_OK=$($PYTHON -c "import sys; print(1 if sys.version_info >= (3,9) else 0)")
echo "    当前: Python $PY_VER"
if [ "$PY_OK" != "1" ]; then
  echo ""
  echo "    ❌ 需要 Python 3.9+ (Flask 3 / akshare 要求)"
  echo ""
  echo "    CentOS 7 安装新 Python:"
  echo "      yum install -y epel-release"
  echo "      yum install -y python39 python39-pip"
  echo "      PYTHON=python3.9 ./start.sh"
  echo ""
  echo "    或用 pyenv / conda 装更高版本"
  exit 1
fi

echo "==> 检查依赖..."
if ! $PYTHON -c "import flask, akshare, requests" 2>/dev/null; then
  echo "    缺失依赖, 安装中..."
  PIP_FLAGS=""
  if $PYTHON -m pip install --help 2>/dev/null | grep -q "break-system-packages"; then
    PIP_FLAGS="--break-system-packages"
  fi
  $PYTHON -m pip install $PIP_FLAGS -q -r requirements.txt
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
