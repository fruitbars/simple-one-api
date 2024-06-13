#!/bin/bash

# 检查是否提供了版本号参数
if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# 定义镜像名称和版本
IMAGE_NAME="simple-one-api"
VERSION="$1"
LATEST="latest"

# 创建并使用 Buildx builder
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

# 构建并推送多平台镜像
docker buildx build --platform linux/amd64,linux/arm64 \
    -t ${IMAGE_NAME}:${VERSION} \
    -t ${IMAGE_NAME}:${LATEST} \
    --push .

# 清理 Buildx builder
docker buildx rm multiarch-builder