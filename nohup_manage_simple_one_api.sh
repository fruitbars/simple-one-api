#!/usr/bin/env bash

set -euo pipefail

# 获取脚本所在目录
DIR="$(cd -- "$(dirname -- "$0")" && pwd)"

# 定义日志文件和PID文件
LOG_FILE="$DIR/simple-one-api.log"
PID_FILE="$DIR/simple-one-api.pid"
CONFIG_PATH="${2:-$DIR/config.json}"

# 启动 simple-one-api 并将输出重定向到日志文件
start() {
  if [ ! -x "$DIR/simple-one-api" ]; then
    echo "错误: $DIR/simple-one-api 不存在或不可执行。" >&2
    exit 1
  fi
  if [ ! -f "$CONFIG_PATH" ]; then
    echo "错误: 配置文件 $CONFIG_PATH 不存在。" >&2
    exit 1
  fi
  if [ -f "$PID_FILE" ]; then
    IFS= read -r PID < "$PID_FILE" || PID=""
    if [[ "$PID" =~ ^[0-9]+$ ]] && ps -p "$PID" > /dev/null 2>&1; then
      echo "simple-one-api 已经在运行，进程ID: $PID"
      exit 1
    else
      echo "发现遗留的PID文件，但没有正在运行的进程。删除PID文件。"
      rm -f "$PID_FILE"
    fi
  fi

  cd "$DIR"
  nohup ./simple-one-api "$CONFIG_PATH" > "$LOG_FILE" 2>&1 &
  PID=$!
  echo "$PID" > "$PID_FILE"
  echo "simple-one-api 已启动，进程ID: $PID，日志文件: $LOG_FILE"
}

# 停止 simple-one-api
stop() {
  if [ -f "$PID_FILE" ]; then
    IFS= read -r PID < "$PID_FILE" || PID=""
    if [[ "$PID" =~ ^[0-9]+$ ]] && ps -p "$PID" > /dev/null 2>&1; then
      kill "$PID"
      echo "simple-one-api 已停止，进程ID: $PID"
      rm -f "$PID_FILE"
    else
      echo "没有正在运行的 simple-one-api 进程，删除遗留的PID文件。"
      rm -f "$PID_FILE"
    fi
  else
    echo "没有找到PID文件，simple-one-api 可能未运行。"
  fi
}

# 重启 simple-one-api
restart() {
  stop
  start
}

# 检查参数
case "${1:-}" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    restart
    ;;
  *)
    echo "使用方法: $0 {start|stop|restart}"
    exit 1
esac
