#!/bin/bash

# 检查是否传入版本号参数
if [ -z "$1" ]; then
  echo "请传入版本号作为参数，例如：./build_and_push.sh v0.8.2"
  exit 1
fi

# 定义变量
IMAGE_NAME="fruitbars/simple-one-api"
TAG=$1

# 构建镜像
docker build -t $IMAGE_NAME:$TAG .

# 打印完成信息
echo "Docker image $IMAGE_NAME:$TAG built"