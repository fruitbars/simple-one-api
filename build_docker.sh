#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

# 检查是否传入版本号参数
if [ "$#" -ne 1 ]; then
  echo "用法: $0 <tag>，例如: $0 v1.0.0" >&2
  exit 1
fi

# 定义变量
IMAGE_NAME="${IMAGE_NAME:-fruitbars/simple-one-api}"
TAG="$1"
TARGET_ARCH="${ARCH:-amd64}"

case "$TARGET_ARCH" in
  amd64|arm64) ;;
  *)
    echo "不支持的 ARCH: $TARGET_ARCH（仅支持 amd64 或 arm64）" >&2
    exit 1
    ;;
esac

make -C "$SCRIPT_DIR" web "build-linux-$TARGET_ARCH"

# 构建镜像
docker build --build-arg "ARCH=$TARGET_ARCH" --tag "$IMAGE_NAME:$TAG" "$SCRIPT_DIR"

# 打印完成信息
echo "Docker image $IMAGE_NAME:$TAG built"
