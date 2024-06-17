#!/bin/bash

# 获取用户输入的平台和架构，默认为当前系统平台和架构
GOOS=${1:-$(go env GOOS)}
GOARCH=${2:-$(go env GOARCH)}

# 设置二进制文件的输出名称
BINARY_NAME="simple-one-api"

# 转到源代码所在目录
cd cmd/simple-one-api

# 编译项目
echo "Building $BINARY_NAME for $GOOS/$GOARCH..."
CGO_ENABLED=0 go build -o $BINARY_NAME

echo "Build completed. Copying the executable to the project root directory..."

# 拷贝编译后的文件到脚本当前目录
cp $BINARY_NAME ../../

# 返回到原始目录
cd - > /dev/null

echo "Build and copy completed successfully!"
