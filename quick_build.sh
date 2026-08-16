#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 获取用户输入的平台和架构，默认为当前系统平台和架构
TARGET_GOOS=${1:-$(go env GOOS)}
TARGET_GOARCH=${2:-$(go env GOARCH)}

# 设置二进制文件的输出名称
BINARY_NAME="simple-one-api"

# 先生成内嵌的 Web 资源，避免服务端携带旧前端
pnpm --dir web install --frozen-lockfile
pnpm --dir web build

OUTPUT_NAME="$BINARY_NAME"
if [[ "$TARGET_GOOS" == "windows" ]]; then
  OUTPUT_NAME+=".exe"
fi

echo "Building $OUTPUT_NAME for $TARGET_GOOS/$TARGET_GOARCH..."
CGO_ENABLED=0 GOOS="$TARGET_GOOS" GOARCH="$TARGET_GOARCH" go build -o "$OUTPUT_NAME"

echo "Build completed."
