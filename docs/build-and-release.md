# 构建与发布

本文档是当前构建、打包和发布流程的唯一说明。

## 环境要求

- Go 1.25 或更新版本
- Node.js 与 pnpm（Web 构建需要）
- Docker（容器镜像需要）
- Wails CLI v2.13.0（桌面端需要）

## 服务端单文件

在仓库根目录执行。Web 生产资源会先编译，再通过 `go:embed` 嵌入可执行文件。

```bash
# 当前平台快速构建，输出到仓库根目录 ./simple-one-api
./quick_build.sh

# 指定目标平台和架构
./quick_build.sh linux amd64

# 全平台发布构建，产物位于 build/
./build.sh --release

# 开发构建；同样会刷新 Web 资源
./build.sh --development

# 查看交叉编译目标
./build.sh --show-platforms
```

不使用脚本时，可以只构建一个目标：

```bash
make web
make build-linux-amd64
```

`make build-<platform>-<arch>` 支持 `darwin-amd64`、`darwin-arm64`、`windows-amd64`、`windows-arm64`、`linux-amd64`、`linux-arm64`、`freebsd-amd64` 和 `freebsd-arm64`。

`--enable-upx` 需要本机已安装 UPX；它只影响 `build.sh --release` 的压缩阶段：

```bash
./build.sh --release --enable-upx
```

## Web 开发

```bash
cd web
pnpm install --frozen-lockfile
pnpm dev
```

提交或发布前运行：

```bash
pnpm typecheck
pnpm test
pnpm build
```

## Wails 桌面端

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
cd cmd/desktop
wails dev
wails build -clean
```

桌面端使用同一套 React Web 资源和 Go 路由，不会额外开放 loopback HTTP 端口。产物位于 `cmd/desktop/build/bin/`。

## Docker

手工构建镜像时，先生成 Linux 服务端：

```bash
make web
make build-linux-amd64
docker build --platform linux/amd64 --build-arg TARGETARCH=amd64 --tag fruitbars/simple-one-api:local .
```

或直接使用脚本。脚本会先刷新 Web、构建 Linux 服务端，再构建镜像；它不会自动推送：

```bash
./build_docker.sh v0.10.3
```

`IMAGE_NAME` 环境变量可以覆盖默认镜像名：

```bash
IMAGE_NAME=registry.example.com/team/simple-one-api ./build_docker.sh v0.10.3
```

默认目标为 `amd64`，构建 ARM64 镜像时设置 `ARCH=arm64`：

```bash
ARCH=arm64 ./build_docker.sh v0.10.3
```

正式版本由 Release workflow 自动发布到 GHCR：

```bash
docker pull ghcr.io/fruitbars/simple-one-api:latest
docker pull ghcr.io/fruitbars/simple-one-api:v0.10.3
```

每个版本是同时包含 `linux/amd64` 和 `linux/arm64` 的多架构 manifest，并附带 provenance 与 SBOM。发布标签包括原始 Git Tag（如 `v1.2.3`）、语义化标签（`1.2.3`、`1.2`、`1`）；稳定版本还会更新 `latest`，带连字符的预发布版本不会覆盖 `latest`。

镜像以非 root 用户 `app` 运行，并通过 `GET /healthz` 执行内置健康检查。该端点不依赖 Web 开关、Provider 配置或 API Key。

## 运行前检查

```bash
./simple-one-api ./config.json
```

生产环境建议为配置目录保留写权限，因为 SQLite 默认数据库会创建在配置文件旁边。若配置文件是只读挂载，请通过 `SIMPLE_ONE_API_DB` 指向可写数据目录。

## GitHub Actions

- [CI workflow](../.github/workflows/ci.yml)：`main` push、Pull Request 和手工触发时运行 Web typecheck/test/build、Go test/bindings/vet、单文件服务端构建、Docker 镜像验证和 macOS Wails 构建。
- [Release workflow](../.github/workflows/release.yml)：推送 `v*` 标签时先运行 Web typecheck/test/build、Go test/bindings/vet 和格式门禁，再构建服务端多平台压缩包、Linux x64、Windows x64、macOS Intel/Apple Silicon 桌面包，以及 amd64/arm64 GHCR 镜像；全部成功后自动创建 GitHub Release 和 `SHA256SUMS`。

发布示例：

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

推荐先把发布提交推送到 `main` 并等待 CI 全绿，再在同一提交上创建带注释的版本标签。发布工作区必须干净，`docs/CHANGELOG.md` 的版本与标签应保持一致。

Release workflow 使用仓库自带的 `GITHUB_TOKEN` 创建 Release 并推送 GHCR，不需要 Docker Hub 或第三方发布密钥。首次发布后，在仓库 Packages 页面打开 `simple-one-api`，将 Package visibility 设置为 Public；后续版本不需要重复设置。Docker Hub 镜像仍由维护者按需手工发布，不属于自动发布链路。
