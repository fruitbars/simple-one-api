<p align="right">
  <strong>中文</strong> | <a href="./README.EN.md">English</a>
</p>

# simple-one-api

用统一网关连接多个大模型 Provider，并提供 OpenAI Chat Completions、OpenAI Responses 和 Anthropic Messages 三种客户端协议，以及内嵌 Web 聊天、可视化配置台和 Wails 桌面端。

项目不负责供应商计费或额度统计。模型名称、价格、免费额度和上游接口以各供应商当前官方文档为准。

## 当前能力

- `/v1/chat/completions`、`/v1/responses`、`/v1/messages`、`/v1/models` 和 Embeddings 接口。
- 多 Provider、多模型、多组凭证的随机、首选、轮询和哈希路由。
- 内嵌 React Web 聊天界面，支持 Markdown、流式输出统计和最多 50 条本地对话历史；生产资源通过 `go:embed` 打进服务端单文件。
- 可视化配置台：编辑基础设置、Provider、模型、凭证、代理和访问密钥，也可以切换到配置源码。
- 可开关的实时日志视图，支持级别筛选、自动跟随、固定容量和敏感信息脱敏。
- SQLite 配置仓库：首次导入 JSON/YAML、校验、保存和运行时原子生效。
- Wails v2 桌面应用；桌面端与 Web 共用界面和 Go 路由，不额外开放 HTTP 端口。
- 全局/Provider 代理、限流、模型别名、翻译和多模态路由。
- Provider/模型粒度的熔断与半开恢复、供应商扩展参数透传，以及思考过程流式展示。
- GitHub Release 自动生成服务端与桌面端产物，并同步发布 amd64/arm64 GHCR 镜像。

支持的 Provider 类型、字段和样例以[配置参考](docs/configuration-reference.md)为准。供应商接入指南仍保留在 [`docs/`](docs/README.md)，其中的额度和模型示例可能过时，使用前请核对官方文档。

## 运行方式

### 直接运行

默认读取可执行文件同目录的 `config.json`，也可以传入 JSON/YAML 路径：

```sh
./simple-one-api
./simple-one-api ./config.json
```

最小 Web 配置：

```json
{
  "server_port": ":9090",
  "enable_web": true,
  "log_level": "info",
  "services": {}
}
```

启动后访问 `http://localhost:9090/` 进入配置台，访问 `http://localhost:9090/chat` 进入聊天界面。兼容路径 `/admin` 也会打开配置台。

### Admin 与 SQLite

- 有主 `api_key` 时，`/api/admin/*` 使用 `Authorization: Bearer <api_key>`。
- 没有主 `api_key` 时，本机 loopback 请求可以进入首次配置；远程访问使用启动日志中的临时 bootstrap token，发布正式 `api_key` 后临时 token 立即失效。
- SQLite 默认位于配置文件旁边（`config.json` → `config.db`），可用 `SIMPLE_ONE_API_DB` 覆盖。
- 配置草稿在 API 边界脱敏密钥；未修改的占位符在发布时恢复原值。
- SQLite 当前没有静态加密，数据库文件尽量使用 `0600` 权限；请限制数据目录访问。

完整流程见[配置参考](docs/configuration-reference.md)。

### Docker

```sh
docker pull ghcr.io/fruitbars/simple-one-api:latest

docker run -d --name simple-one-api -p 9090:9090 \
  -v /absolute/path/config.json:/app/config.json:ro \
  -v /absolute/path/data:/app/data \
  -e SIMPLE_ONE_API_DB=/app/data/config.db \
  ghcr.io/fruitbars/simple-one-api:latest
```

正式环境建议将 `latest` 替换为固定版本，例如 `v0.10.1`。镜像同时支持 `linux/amd64` 和 `linux/arm64`，内置 `/healthz` 健康检查。仓库内的 `docker-compose.yml` 默认挂载当前目录的 `config.json` 和 `data/`；配置文件只读挂载时，SQLite 必须指向可写数据目录。

其他部署方式：[systemd](docs/startup/systemd_startup.md) · [nohup](docs/startup/nohup_startup.md)。

## 构建：应该用哪个脚本？

| 目标 | 命令 | 产物 |
| --- | --- | --- |
| 当前平台快速构建 | `./quick_build.sh` | 根目录 `simple-one-api` |
| 指定平台构建 | `./quick_build.sh linux amd64` | 根目录目标二进制 |
| 多平台发布 | `./build.sh --release` | `build/` 下二进制和压缩包 |
| 开发构建 | `./build.sh --development` | `build/` 下各平台二进制 |
| Docker 镜像 | `./build_docker.sh vX.Y.Z` | 本地镜像（不推送） |

构建要求 Go 1.25+、Node.js 和 pnpm。所有脚本都会先构建 `web/`，避免把旧前端嵌入服务端。完整说明见[构建与发布](docs/build-and-release.md)。

Windows 可运行 `quick_build.bat`，也支持传入 `GOOS GOARCH` 参数。

### Web 开发

```sh
cd web
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
```

构建结果写入 `internal/webui/dist/`，随后执行 Go 构建即可得到单文件服务端。

### Wails 桌面端

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
cd cmd/desktop
wails dev
wails build -clean
```

产物位于 `cmd/desktop/build/bin/`。桌面端说明见 [`cmd/desktop/README.md`](cmd/desktop/README.md)。

## API 示例

```sh
curl http://localhost:9090/v1/models \
  -H 'Authorization: Bearer your-gateway-key'

curl http://localhost:9090/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-gateway-key' \
  -d '{"model":"random","messages":[{"role":"user","content":"你好"}]}'

curl http://localhost:9090/v1/responses \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-gateway-key' \
  -d '{"model":"random","input":"你好"}'

curl http://localhost:9090/v1/messages \
  -H 'Content-Type: application/json' \
  -H 'x-api-key: your-gateway-key' \
  -H 'anthropic-version: 2023-06-01' \
  -d '{"model":"random","max_tokens":256,"messages":[{"role":"user","content":"你好"}]}'
```

OpenAI 兼容 SDK 可将 `base_url` 指向 `http://host:9090/v1`。Codex 使用 Responses 协议；Claude Code 使用 Anthropic Messages 协议，具体限制见[配置参考](docs/configuration-reference.md)。

## 配置和接入文档

- [文档索引](docs/README.md)
- [配置参考（字段、Provider、Admin、SQLite）](docs/configuration-reference.md)
- [构建与发布](docs/build-and-release.md)
- [架构说明](docs/architecture-v1.md)
- [更新日志](docs/CHANGELOG.md)
- [样例配置](samples/)

Provider 的申请/接入文档是历史辅助材料；模型、额度、URL 和认证方式可能变化，请以官方文档为准。

## 发布产物

- [GitHub Releases](https://github.com/fruitbars/simple-one-api/releases)：服务端多平台归档、桌面端包和 `SHA256SUMS`。
- [GHCR 镜像](https://github.com/fruitbars/simple-one-api/pkgs/container/simple-one-api)：`linux/amd64`、`linux/arm64` 多架构镜像。
- 推送 `v*` Tag 后，两类产物由同一个 Release workflow 构建；只有所有平台和容器镜像都成功后才创建 GitHub Release。

## 贡献

欢迎提交 Issue 和 Pull Request。提交前请运行 `go test ./...`、`go vet ./...`，以及 `cd web && pnpm typecheck && pnpm test && pnpm build`。
