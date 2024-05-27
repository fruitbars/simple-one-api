#!/bin/bash

# 设置目标操作系统和架构
export GOOS=windows
export GOARCH=amd64

# 设置输出目录
OUTPUT_DIR="build/windows"
mkdir -p $OUTPUT_DIR

# 检查是否传入 --rebuild 参数
if [ "$1" == "--rebuild" ]; then
    echo "正在进行全量重新编译 Windows 版本..."
    go build -a -o $OUTPUT_DIR/simple-one-api.exe main.go
else
    echo "正在编译 Windows 版本..."
    go build -o $OUTPUT_DIR/simple-one-api.exe main.go
fi

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "Windows 版本编译成功，已移动到 $OUTPUT_DIR 目录"
else
    echo "Windows 版本编译失败"
    exit 1
fi
