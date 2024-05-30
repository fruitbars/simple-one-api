#!/bin/bash

# 设置目标操作系统和架构
export GOOS=linux
export GOARCH=amd64

# 设置输出目录
OUTPUT_DIR="build/simple-one-api-linux"
mkdir -p $OUTPUT_DIR

# 检查是否传入 --rebuild 参数
if [ "$1" == "--rebuild" ]; then
    echo "正在进行全量重新编译 Linux 版本..."
    go build -a -o $OUTPUT_DIR/simple-one-api main.go
else
    echo "正在编译 Linux 版本..."
    go build -o $OUTPUT_DIR/simple-one-api main.go
fi

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "Linux 版本编译成功，已移动到 $OUTPUT_DIR 目录"
else
    echo "Linux 版本编译失败"
    exit 1
fi
