<p align="right">
  <strong>English</strong> | <a href="./README.md">中文</a>
</p>

# simple-one-api

Expose multiple LLM providers through one gateway with OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages client protocols, plus embedded Web chat, visual configuration, and a Wails desktop app.

This project does not track provider billing or quotas. Model names, prices, free tiers, and upstream endpoints should always be checked against the provider's current official documentation.

## Highlights

- `/v1/chat/completions`, `/v1/responses`, `/v1/messages`, `/v1/models`, and Embeddings endpoints.
- Multiple providers, models, and credential sets with random, first, round-robin, or hash routing.
- Embedded React Web chat with Markdown, streaming metrics, and up to 50 local conversations; production assets are compiled into the server binary with `go:embed`.
- Visual configuration for system settings, providers, models, credentials, proxies, and access keys, plus a source editor.
- Optional real-time logs with level filters, follow mode, bounded memory, and secret redaction.
- SQLite configuration repository with JSON/YAML import, validation, save, and atomic runtime activation.
- Wails v2 desktop app sharing the same UI and Go routes without opening a local HTTP port.
- Global and per-provider proxies, rate limits, model aliases, translation, and multimodal routing.

See the [configuration reference](docs/configuration-reference.md) for the authoritative provider list, fields, and samples. Historical provider guides remain under [`docs/`](docs/README.md); quota and model examples may be outdated, so verify them with the provider.

## Run

### Server

The server reads `config.json` by default. Pass a JSON or YAML path to override it:

```sh
./simple-one-api
./simple-one-api ./config.json
```

Minimal Web configuration:

```json
{
  "server_port": ":9090",
  "enable_web": true,
  "log_level": "info",
  "services": {}
}
```

Open `http://localhost:9090/` for configuration and `http://localhost:9090/chat` for chat. The compatibility path `/admin` also opens configuration.

### Admin and SQLite

- With a primary `api_key`, `/api/admin/*` requires `Authorization: Bearer <api_key>`.
- Without a primary `api_key`, loopback requests can perform first-run setup. Remote users unlock setup with the temporary bootstrap token printed at startup; publishing a permanent `api_key` immediately invalidates that token.
- SQLite defaults to the configuration file's directory and basename (`config.json` → `config.db`). Override it with `SIMPLE_ONE_API_DB`.
- Draft responses mask secrets, and unchanged placeholders are restored when a revision is published.
- SQLite data is not encrypted at rest. The database is created with `0600` permissions when possible; restrict access to its directory.

See the [configuration reference](docs/configuration-reference.md) for the complete workflow.

### Docker

```sh
docker run -d --name simple-one-api -p 9090:9090 \
  -v /absolute/path/config.json:/app/config.json:ro \
  -v /absolute/path/data:/app/data \
  -e SIMPLE_ONE_API_DB=/app/data/config.db \
  fruitbars/simple-one-api
```

Use `docker-compose.yml` as a template and replace the configuration and data paths. If the configuration is mounted read-only, the SQLite database must point to a writable directory.

Other deployment options: [systemd](docs/startup/systemd_startup.md) · [nohup](docs/startup/nohup_startup.md).

## Build: which script should I use?

| Goal | Command | Output |
| --- | --- | --- |
| Fast build for this platform | `./quick_build.sh` | Root-level `simple-one-api` |
| Build one target | `./quick_build.sh linux amd64` | Root-level target binary |
| Multi-platform release | `./build.sh --release` | Binaries and archives under `build/` |
| Development build | `./build.sh --development` | Platform binaries under `build/` |
| Docker image | `./build_docker.sh vX.Y.Z` | Local image only; it is not pushed |

Building requires Go 1.25+, Node.js, and pnpm. Each entry point builds `web/` before compiling Go so stale frontend assets are not embedded. See [Build and Release](docs/build-and-release.md) for the complete matrix.

On Windows, run `quick_build.bat`; optional `GOOS GOARCH` arguments are supported.

### Web development

```sh
cd web
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
```

The Web build writes to `internal/webui/dist/`. A subsequent Go build produces a single-file server.

### Wails desktop app

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
cd cmd/desktop
wails dev
wails build -clean
```

Artifacts are written to `cmd/desktop/build/bin/`. See [`cmd/desktop/README.md`](cmd/desktop/README.md) for desktop details.

## API examples

```sh
curl http://localhost:9090/v1/models \
  -H 'Authorization: Bearer your-gateway-key'

curl http://localhost:9090/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-gateway-key' \
  -d '{"model":"random","messages":[{"role":"user","content":"Hello"}]}'

curl http://localhost:9090/v1/responses \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-gateway-key' \
  -d '{"model":"random","input":"Hello"}'

curl http://localhost:9090/v1/messages \
  -H 'Content-Type: application/json' \
  -H 'x-api-key: your-gateway-key' \
  -H 'anthropic-version: 2023-06-01' \
  -d '{"model":"random","max_tokens":256,"messages":[{"role":"user","content":"Hello"}]}'
```

OpenAI-compatible SDKs can set `base_url` to `http://host:9090/v1`. Codex uses the Responses wire protocol, while Claude Code uses Anthropic Messages; see the [configuration reference](docs/configuration-reference.md) for limits.

## Documentation

- [Documentation index](docs/README.md)
- [Configuration reference: fields, providers, Admin, and SQLite](docs/configuration-reference.md)
- [Build and release](docs/build-and-release.md)
- [Architecture](docs/architecture-v1.md)
- [Changelog](docs/CHANGELOG.md)
- [Configuration samples](samples/)

Provider setup guides are retained as historical aids. Models, quotas, URLs, and authentication methods may change; prefer official provider documentation.

## Contributing

Issues and pull requests are welcome. Before submitting, run `go test ./...`, `go vet ./...`, and `cd web && pnpm typecheck && pnpm test && pnpm build`.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=fruitbars/simple-one-api&type=Date)](https://star-history.com/#fruitbars/simple-one-api&Date)
