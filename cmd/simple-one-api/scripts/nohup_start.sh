#!/bin/bash

# 获取脚本所在目录
DIR=$(dirname "$0")

# 定义日志文件
LOG_FILE="$DIR/logfile.log"

# 进入程序所在目录
cd $DIR

# 启动 simple-one-api 并将输出重定向到日志文件
nohup ./simple-one-api > $LOG_FILE 2>&1 &

# 获取进程ID
PID=$!

# 输出启动信息
echo "simple-one-api 已启动，进程ID: $PID，日志文件: $LOG_FILE"
