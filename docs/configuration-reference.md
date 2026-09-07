# 配置参考

JSON 和 YAML 都可以作为启动时的导入格式。启动后 SQLite 是运行时配置仓库；配置台每次保存都会写入规范化快照并原子更新运行时配置。

## 最小 Web 配置

```json
{
  "server_port": ":9090",
  "enable_web": true,
  "log_level": "info",
  "load_balancing": "random",
  "circuit_breaker": {
    "enabled": true,
    "failure_threshold": 5,
    "recovery_timeout_seconds": 30,
    "half_open_max_requests": 1
  },
  "statistics": {
    "enabled": true,
    "retention_days": 30
  },
  "services": {}
}
```

首次没有 `api_key` 时，本机可以直接打开 `/` 或 `/admin`。远程访问需要启动日志中的临时 bootstrap token，进入后台后在“基础设置”填写正式 `api_key` 并保存配置。

## 顶层字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `server_port` | string | `:9090` 或 `127.0.0.1:9090`，端口范围 1–65535。变更需要重启。 |
| `enable_web` | boolean | 是否启用内嵌 Web 与 Admin，变更需要重启。 |
| `api_key` | string | 网关主密钥，同时保护 `/api/admin/*` 和需要鉴权的 OpenAI 兼容接口。 |
| `api_keys` | array | 可选的细粒度客户端密钥与模型权限。 |
| `debug` | boolean | 调试模式，变更需要重启。 |
| `log_level` | string | `debug`、`info`、`warn`、`error`、`prodj` 等兼容值，变更需要重启。 |
| `load_balancing` | string | `random`、`first`、`round_robin`、`hash`。 |
| `circuit_breaker` | object | Provider-Key-模型粒度的熔断与自动恢复；默认连续失败 5 次后暂停 30 秒，并放行 1 个半开探测请求。 |
| `statistics` | object | 轻量使用统计；默认启用并保留 30 天，保留期范围为 1–3650 天。 |
| `services` | object | Provider 配置，键名是支持的服务类型。 |
| `proxy` | object | 全局 HTTP/HTTPS/SOCKS5 代理。 |
| `multi_content_models` | string[] | 允许多模态内容的模型匹配列表。 |
| `model_redirect` | object | 全局模型重定向。 |
| `params_range` | object | 模型参数范围。 |
| `translation` | object | 翻译功能和并发设置。 |

## Provider 配置

当前支持的服务类型：

`openai`、`azure`、`deepseek`、`zhipu`、`groq`、`ollama`、`gemini`、`claude`、`qianfan`、`hunyuan`、`xinghuo`、`minimax`、`huoshan`、`dashscope`、`bailian`、`dify`、`vertexai`。

Coze（含 v2/v3）和百度 AgentBuilder 已停止支持；包含这些旧 Provider 的草稿会在校验时给出错误。

## 客户端协议

网关同时提供三种客户端入口，均复用相同的 Provider、模型路由、限流和鉴权配置：

- OpenAI Chat Completions：`POST /v1/chat/completions`
- OpenAI Responses：`POST /v1/responses`，可供 Codex 自定义 Provider 使用
- Anthropic Messages：`POST /v1/messages`，可供 Claude Code 使用
- OpenAI 模型列表：`GET /v1/models`；单模型查询：`GET /v1/models/:model`

Responses 与 Messages 入口支持文本、图片、函数工具定义、工具调用和工具结果。使用 Chat Completions 上游时，流式请求会实时消费其 SSE 并转换为对应客户端协议事件；选择 Responses 上游时，请求字段和响应事件直接透传。客户端断开会取消上游请求。不支持的有状态会话续接或内容类型会返回明确的协议错误，不会静默忽略。

Chat Completions 会将 SDK 未建模的顶层 JSON 字段原样透传给 OpenAI 兼容上游，例如 DashScope/Qwen 的 `enable_thinking`。网关规范化后的 `model`、`messages` 和流式选项优先，客户端不能借此绕过模型路由。内置 Chat 的“思考”开关会同时发送 `reasoning_effort`、`enable_thinking` 和 `chat_template_kwargs.enable_thinking`；上游返回的 `reasoning_content` 或 `reasoning` 会与正文分离并实时展示。

熔断状态以 Provider 稳定 `id` 和客户端模型为粒度。达到 `failure_threshold` 后，该组合在 `recovery_timeout_seconds` 内不会参与负载均衡；等待结束后最多放行 `half_open_max_requests` 个并发探测，任一成功会关闭熔断，失败则重新开始恢复计时。设置 `circuit_breaker.enabled` 为 `false` 可以关闭此行为。

所有 `/v1/*` POST 请求的请求体上限为 8 MiB。网关主密钥支持 `Authorization: Bearer <key>`；Anthropic 客户端也可以使用 `x-api-key: <key>`。

`services.<type>` 是数组，每个条目可以包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | Provider 稳定 ID。缺少时自动生成 `<type>-<序号>` 并在发布时持久化。 |
| `provider` | string | Provider 标识，通常与服务类型相同。 |
| `upstream_protocol` | string | 上游接口协议：`auto`、`chat_completions`、`responses`、`anthropic_messages`。省略时默认为 `auto`。 |
| `enabled` | boolean | 是否参与模型路由。 |
| `models` | string[] | 聊天模型列表，会自动去空格、去重。 |
| `embedding_models` | string[] | Embedding 模型列表。 |
| `server_url` | string | 上游 HTTP(S) 或 WebSocket 地址。 |
| `credentials` | object | Provider 凭证。不同服务需要的字段不同。 |
| `credential_list` | object[] | 多组轮换凭证；每项支持 `id`、`name`、`enabled`、服务商凭证字段、独立 `limit` 和按模型的 `model_limits`。 |
| `model_map` | object | Provider 内部模型别名映射。 |
| `model_redirect` | object | Provider 内部模型重定向。 |
| `limit` / `embedding_limit` | object | `qps`、`qpm`、`rpm`、`tpm`、`concurrency`、`timeout`，数值不能为负；设置多项时组合生效。 |
| `model_limits` | object | 按模型设置独立组合限流；键为模型名，值支持同样的限流字段。 |
| `use_proxy` | boolean | 覆盖全局代理策略。 |
| `timeout` | number | 单次请求超时秒数。 |

`upstream_protocol` 用于将服务商类型与实际上游 HTTP 协议解耦。`auto` 保持现有按 Provider 适配器路由的行为；显式选择 `chat_completions`、`responses` 或 `anthropic_messages` 时，网关会使用对应协议发送请求。

`credential_list` 可以作为 Provider 的 API Key 号池使用：每组凭证通常至少包含 `api_key`，也可以配置稳定 `id`、显示 `name`、`enabled` 和独立 `limit`。凭证以全局 `load_balancing` 策略作为基础顺序，再优先选择能容纳当前请求且剩余 TPM 较高的 Key；在尚未写出响应且遇到可恢复的鉴权、限流、网络或上游服务错误时，会自动尝试池内下一个健康凭证。熔断状态按 `Provider ID + Credential ID + 模型` 独立记录，单个 Key 冷却不会拖停同池其他 Key。

Provider 的 `limit`（聊天）或 `embedding_limit`（Embedding）与当前 Key 的 `limit` 会叠加执行。QPS、QPM、RPM、TPM 和并发数不是互斥选项，配置了几项就同时满足几项；Key 达到限制时可以切换到池内其他 Key，Provider 总限制达到后不会通过换 Key 绕过。`timeout` 是等待限流额度的最长秒数，超时返回 HTTP 429。

模型还可以配置独立的 `model_limits`。Provider 上的 `model_limits.<model>` 是该 Provider 下模型的共享限制；号池凭证中的 `model_limits.<model>` 是“当前 API Key + 当前模型”的限制。一次请求会按顺序叠加 Key 总限制、Key-模型限制、Provider-模型限制和 Provider 总限制。Provider 层或 Provider-模型层达到上限时不会通过切换 Key 绕过；Key 层达到上限时，号池仍可以切换到其他 Key。

例如：

```json
{
  "models": ["deepseek-chat", "deepseek-reasoner"],
  "model_limits": {
    "deepseek-chat": {"tpm": 100000, "concurrency": 5}
  },
  "credential_list": [
    {
      "id": "key-a",
      "enabled": true,
      "api_key": "sk-...",
      "limit": {"qps": 2},
      "model_limits": {
        "deepseek-reasoner": {"tpm": 20000}
      }
    }
  ]
}
```

TPM 在请求发往上游前预占：聊天请求按消息、工具定义和最大输出 Token 估算，Embedding 按输入体积估算。该值是面向限流的保守近似，不是供应商 tokenizer 的精确计费结果；单次估算已经超过 TPM 上限时会立即返回 429。QPM 与 RPM 都保留为一分钟请求数窗口，用于兼容不同上游配置命名；若两者同时填写，会按两个窗口共同约束。

容量调度状态保存在 Go 进程内存中，以 60 秒滚动窗口清理预留。上游返回 429 时，当前 `Key + 模型` 默认冷却 30 秒；调度器会综合 TPM 窗口和冷却截止时间计算下一次可用时间。进程重启会清空这些运行时预留，这套数据用于本地调度，不代表供应商账户余额或账单。

启用的 Provider 至少要有聊天或 Embedding 模型；`qianfan`、`hunyuan`、`deepseek`、`zhipu`、`minimax`、`huoshan`、`gemini`、`groq`、`xinghuo` 等存在默认模型映射的服务可以省略 `models`。

## 代理

```json
{
  "proxy": {
    "strategy": "default",
    "type": "http",
    "http_proxy": "http://127.0.0.1:7890",
    "https_proxy": "http://127.0.0.1:7890",
    "timeout": 30
  }
}
```

`strategy` 支持 `disabled`、`default`、`all`、`force_all`。启用代理时必须同时提供类型和对应地址。代理 URL 中的用户名和密码会在 Admin 接口脱敏。

## SQLite 与文件配置

- 默认数据库：配置文件同目录、同名 `.db`，例如 `config.json` 对应 `config.db`。
- 覆盖路径：设置 `SIMPLE_ONE_API_DB=/data/simple-one-api/config.db`。
- 首次启动导入文件配置；文件 checksum 变化时导入新 revision。
- 权威来源采用兼容模式：运行期间以 SQLite 的 active revision 为准；重启时，如果启动文件 checksum 发生变化，文件会作为新的 active revision 导入，因此运维人员仍可通过显式修改启动文件覆盖后台最近发布的版本。
- 未知 JSON/YAML 字段会被保留，表单编辑不会清除它们。
- SQLite 当前未做静态加密，数据库文件权限尽量设置为 `0600`；生产环境应限制数据目录权限。

## 使用统计

- 配置台“使用统计”提供预设及自定义时间范围、上一周期对比，并可按 Provider、模型、协议、访问密钥和状态组合筛选。
- 摘要包含输入/输出 Token、Usage 完整率、P50/P95 延迟、流式 TTFT 和输出 Token 速率；Provider/模型分布会分别显示 Usage 完整率。
- 每个 `/v1/*` POST 请求都会返回 `X-Request-ID`。数据库只记录请求 ID、时间、协议、Access Key 指纹、模型、Provider、状态码、延迟和上游返回的 Token 数。
- 不记录原始 API Key、请求或响应正文、IP、User-Agent。Access Key 只保存不可逆 SHA-256 短指纹。
- 上游未返回 Usage 时，Token 字段保存为 `NULL`，不会估算成 0。支持输入、输出、缓存输入、缓存写入、推理和总 Token 字段。
- 写入使用有界异步队列、批量事务和 SQLite WAL，不阻塞推理响应；队列或数据库繁忙造成的丢弃数会显示在统计页底部。
- `statistics.enabled` 和 `statistics.retention_days` 保存后立即生效。过期记录会自动从同一个 SQLite 数据库的 `request_stats` 表清理。
- 管理聚合接口为 `GET /api/admin/statistics/overview?from=<RFC3339>&to=<RFC3339>&bucket=hour|day`，可选筛选参数为 `provider`、`model`、`protocol`、`access_key` 和 `status=success|failure`。
- CSV 接口为 `GET /api/admin/statistics/export`，接受与聚合接口相同的时间和筛选参数，并使用现有 Admin 鉴权。
- 号池容量接口为 `GET /api/admin/capacity`，返回当前启用 Provider 的 Key-模型 TPM 上限、预留、剩余、可用状态、冷却截止和预计恢复时间；响应不包含上游密钥。

## 桌面本地网关

Wails App 启动时会在 `127.0.0.1:<server_port>` 启动与服务端相同的 API 网关，默认地址为 `http://127.0.0.1:9090`。本地 OpenAI 兼容客户端可使用 `http://127.0.0.1:9090/v1` 作为 Base URL，并通过网关主 `api_key` 鉴权。关闭 App 会停止网关并释放端口；如果端口已被其他进程占用，桌面 UI 仍可运行，但外部客户端无法连接该 App 的网关。

## 配置台保存流程

1. 打开 `/` 或 `/admin`，可视化表单是默认入口。
2. 修改只存在于浏览器草稿。
3. 点击“校验配置”检查端口、Provider、模型、代理、限流等规则。
4. 点击“保存配置”写入 SQLite 快照并立即更新运行时配置。

`server_port`、`enable_web`、`debug`、`log_level` 变更会标记为需要重启。Provider、模型、凭证、代理和负载均衡通常会在保存后立即更新。底层仍保留 revision API 供兼容和运维使用，但当前配置台不提供版本历史界面。

## 实时日志与聊天历史

- 配置台“实时日志”默认只在页面开启时每秒拉取一次；关闭开关后停止请求。
- 内存中最多保留 500 条脱敏日志，不记录结构化请求正文和 Zap 字段。
- Web 与桌面聊天历史保存在当前浏览器/WebView 的 `localStorage`，不会上传服务端。
- 最多保存 50 个会话、约 4 MiB；清理浏览器站点数据或应用 WebView 数据会删除这些历史。
- Access Key 只保存在当前会话的 `sessionStorage`，不会随聊天历史持久化。

## 配置样例

- [Web 空配置](../samples/config_web.json)
- [OpenAI 兼容配置](../samples/config.json)
- [Gemini](../samples/config_gemini.json)
- [Qianfan](../samples/config_qianfan.json)
- [Ollama](../samples/config_ollama.json)
- [Provider 接入指南目录](./README.md)
