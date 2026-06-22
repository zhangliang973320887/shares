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

# 选项解析
DAEMON=0
ACTION=start
LOG_FILE=${LOG_FILE:-app.log}
PID_FILE=${PID_FILE:-app.pid}
for arg in "$@"; do
  case "$arg" in
    -d|--daemon) DAEMON=1 ;;
    stop)        ACTION=stop ;;
    restart)     ACTION=restart ;;
    status)      ACTION=status ;;
    -h|--help)
      cat <<USAGE
用法: $0 [选项] [命令]

选项:
  -d, --daemon    后台运行 (nohup, 日志 → $LOG_FILE, pid → $PID_FILE)

命令:
  (无)            前台启动
  stop            停止后台进程
  restart         重启 (=stop + 前台/后台)
  status          查看进程

环境变量:
  PORT=5000  HOST=127.0.0.1  LOG_FILE=app.log  PID_FILE=app.pid

示例:
  ./start.sh                       前台启动
  ./start.sh -d                    后台启动
  HOST=0.0.0.0 ./start.sh -d       后台+公网监听
  ./start.sh stop                  停止后台
  ./start.sh status                查看
  ./start.sh restart -d            重启到后台
USAGE
      exit 0
      ;;
  esac
done

# stop/status 不需编译
if [ "$ACTION" = "stop" ] || [ "$ACTION" = "status" ]; then
  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      if [ "$ACTION" = "status" ]; then
        echo "✓ 运行中, PID=$PID"
        ps -p "$PID" -o pid,etime,cmd 2>/dev/null || true
        exit 0
      else
        kill "$PID" && rm -f "$PID_FILE"
        echo "✓ 已停 PID=$PID"
        exit 0
      fi
    fi
  fi
  if [ "$ACTION" = "status" ]; then
    echo "✗ 未运行"
  else
    echo "✗ 未发现 pid 文件 ($PID_FILE)"
  fi
  exit 0
fi

# restart: 先停后台
if [ "$ACTION" = "restart" ] && [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  kill "$PID" 2>/dev/null || true
  rm -f "$PID_FILE"
  sleep 1
fi

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
  echo "==> Go 环境诊断:"
  echo "    go version: $(go version)"
  echo "    GOMOD: $(go env GOMOD)"
  echo "    GO111MODULE: $(go env GO111MODULE)"
  echo "    GOPATH: $(go env GOPATH)"
  echo "    PWD: $(pwd)"
  if [ "$(go env GOMOD)" = "/dev/null" ] || [ -z "$(go env GOMOD)" ]; then
    echo "❌ Go 未在模块模式 (GOMOD 为空)"
    echo "    可能原因: 当前目录在 GOPATH/src 之下,Go 用了 GOPATH 模式"
    echo "    解决: 移出 GOPATH/src 或 export GOFLAGS='-mod=mod'"
    exit 1
  fi
  echo "==> go mod tidy..."
  go mod tidy
  echo "==> 编译..."
  go build -mod=mod -o valuation .
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

if [ "$DAEMON" = "1" ]; then
  PORT=$PORT HOST=$HOST nohup "$BIN" > "$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"
  sleep 1
  if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "✓ 后台运行 PID=$(cat $PID_FILE)"
    echo "    日志: tail -f $LOG_FILE"
    echo "    停服: $0 stop"
    echo "    状态: $0 status"
  else
    echo "❌ 启动失败,看日志:"
    tail -20 "$LOG_FILE"
    rm -f "$PID_FILE"
    exit 1
  fi
else
  echo "    Ctrl+C 退出 (后台: $0 -d)"
  trap 'echo ""; echo "==> 停服"; exit 0' INT TERM
  PORT=$PORT HOST=$HOST exec "$BIN"
fi
