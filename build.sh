#!/bin/bash

# 默认禁用 UPX 压缩
use_upx=0
# 设置默认构建选项为 release
build_option="release"
# 默认不删除构建目录
clean_up=0

# 获取命令行参数
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --enable-upx)
            use_upx=1
            shift
            ;;
        --show-platforms)
            echo "Available platforms:"
            echo "  - darwin-amd64"
            echo "  - darwin-arm64"
            echo "  - windows-amd64"
            echo "  - windows-arm64"
            echo "  - linux-amd64"
            echo "  - linux-arm64"
            echo "  - freebsd-amd64"
            echo "  - freebsd-arm64"
            exit 0
            ;;
        --development)
            build_option="dev"
            shift
            ;;
        --release)
            build_option="release"
            shift
            ;;
        --clean-up)
            clean_up=1
            shift
            ;;
        *)
            echo "Invalid option: $1"
            echo "Usage: $0 [--enable-upx] [--show-platforms] [--development | --release] [--clean-up]"
            exit 1
            ;;
    esac
done

# 根据指定的构建选项执行相应操作
case $build_option in
    dev)
        echo "Building (Development)..."
        make dev use_upx=$use_upx clean_up=$clean_up
        ;;
    release)
        echo "Building and Releasing..."
        make release use_upx=$use_upx clean_up=$clean_up
        ;;
    *)
        echo "No build option specified or invalid option. Exiting."
        echo "Use --development for dev build or --release for release build."
        exit 1
        ;;
esac

echo "Build script finished."