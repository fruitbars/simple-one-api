#!/bin/bash

# 定义变量
SERVICE_NAME="simple-one-api"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"
WORKING_DIRECTORY="/path/to/your/application"
EXEC_START="$WORKING_DIRECTORY/simple-one-api"
LOG_FILE="$WORKING_DIRECTORY/simple-one-api.log"

# 创建 systemd 服务单元文件内容
SERVICE_CONTENT="[Unit]
Description=Simple One API Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$WORKING_DIRECTORY
ExecStart=$EXEC_START
Restart=on-failure
StandardOutput=append:$LOG_FILE
StandardError=append:$LOG_FILE

[Install]
WantedBy=multi-user.target"

# 检查工作目录和可执行文件是否存在
if [ ! -d "$WORKING_DIRECTORY" ]; then
  echo "错误: 工作目录 $WORKING_DIRECTORY 不存在。请检查路径。"
  exit 1
fi

if [ ! -x "$EXEC_START" ]; then
  echo "错误: 可执行文件 $EXEC_START 不存在或不可执行。请检查路径。"
  exit 1
fi

# 创建服务单元文件
echo "创建服务单元文件 $SERVICE_FILE"
echo "$SERVICE_CONTENT" | sudo tee $SERVICE_FILE > /dev/null

# 重新加载 systemd 配置
echo "重新加载 systemd 配置"
sudo systemctl daemon-reload

# 启动并启用服务
echo "启动 $SERVICE_NAME 服务"
sudo systemctl start $SERVICE_NAME

echo "启用 $SERVICE_NAME 服务在启动时自动运行"
sudo systemctl enable $SERVICE_NAME

echo "$SERVICE_NAME 服务已安装并启动"
