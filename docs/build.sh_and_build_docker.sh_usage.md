# README

## build.sh使用说明

### 默认发布构建
运行以下命令将默认进行发布构建（不启用 UPX 压缩且不删除构建目录）：

```bash
./build.sh
```

### 启用 UPX 压缩的发布构建，并删除构建目录
如果你想要启用 UPX 压缩并进行发布构建，同时在压缩后删除构建目录，可以运行：

```bash
./build.sh --enable-upx --clean-up
```

### 进行开发构建
运行以下命令进行开发构建（不启用 UPX 压缩且不删除构建目录）：

```bash
./build.sh --development
```

### 显示支持的平台
如果需要查看支持的平台列表，可以使用：

```bash
./build.sh --show-platforms
```


## build_docker.sh使用说明

1. **赋予执行权限**：确保脚本具有执行权限。

   ```sh
   chmod +x build_docker.sh
   ```

2. **运行脚本**：运行脚本并传递版本号参数（例如 `v0.5`）。

   ```sh
   ./build_docker.sh v0.5
   ```