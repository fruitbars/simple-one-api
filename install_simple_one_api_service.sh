#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "用法: $0 <应用目录> [配置文件路径]" >&2
  exit 1
fi

SERVICE_NAME="simple-one-api"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
APPLICATION_DIRECTORY="$1"
if [ ! -d "$APPLICATION_DIRECTORY" ]; then
  echo "错误: 工作目录 $APPLICATION_DIRECTORY 不存在。" >&2
  exit 1
fi
WORKING_DIRECTORY="$(cd -- "$APPLICATION_DIRECTORY" && pwd)"
EXEC_START="$WORKING_DIRECTORY/simple-one-api"
CONFIG_INPUT="${2:-$WORKING_DIRECTORY/config.json}"

if [ ! -x "$EXEC_START" ]; then
  echo "错误: 可执行文件 $EXEC_START 不存在或不可执行。" >&2
  exit 1
fi
if [ ! -f "$CONFIG_INPUT" ]; then
  echo "错误: 配置文件 $CONFIG_INPUT 不存在。" >&2
  exit 1
fi
CONFIG_DIRECTORY="$(cd -- "$(dirname -- "$CONFIG_INPUT")" && pwd)"
CONFIG_PATH="$CONFIG_DIRECTORY/$(basename -- "$CONFIG_INPUT")"
DATABASE_PATH="$WORKING_DIRECTORY/config.db"

systemd_quote() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '"%s"' "$value"
}

WORKING_DIRECTORY_UNIT="$(systemd_quote "$WORKING_DIRECTORY")"
EXEC_START_UNIT="$(systemd_quote "$EXEC_START")"
CONFIG_PATH_UNIT="$(systemd_quote "$CONFIG_PATH")"
DATABASE_ENV_UNIT="$(systemd_quote "SIMPLE_ONE_API_DB=$DATABASE_PATH")"

SERVICE_CONTENT="[Unit]
Description=Simple One API Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$WORKING_DIRECTORY_UNIT
ExecStart=$EXEC_START_UNIT $CONFIG_PATH_UNIT
Restart=on-failure
RestartSec=3
StandardOutput=journal
StandardError=journal
Environment=$DATABASE_ENV_UNIT

[Install]
WantedBy=multi-user.target"

# 创建服务单元文件
echo "创建服务单元文件 $SERVICE_FILE"
printf '%s\n' "$SERVICE_CONTENT" | sudo tee "$SERVICE_FILE" > /dev/null

echo "重新加载 systemd 配置"
sudo systemctl daemon-reload

echo "启用并启动 $SERVICE_NAME 服务"
sudo systemctl enable --now "$SERVICE_NAME"

echo "$SERVICE_NAME 服务已安装并启动"
