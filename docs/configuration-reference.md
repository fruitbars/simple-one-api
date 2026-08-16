# 配置参考

JSON 和 YAML 都可以作为启动时的导入格式。启动后 SQLite 是运行时配置仓库；配置台每次保存都会写入规范化快照并原子更新运行时配置。

## 最小 Web 配置

```json
{
  "server_port": ":9090",
  "enable_web": true,
  "log_level": "info",
  "load_balancing": "random",
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

Responses 与 Messages 入口支持文本、图片、函数工具定义、工具调用和工具结果。流式请求会实时消费上游 Chat Completions SSE，并转换为对应协议事件；客户端断开会取消上游请求。不支持的有状态会话续接或内容类型会返回明确的协议错误，不会静默忽略。

所有 `/v1/*` POST 请求的请求体上限为 8 MiB。网关主密钥支持 `Authorization: Bearer <key>`；Anthropic 客户端也可以使用 `x-api-key: <key>`。

`services.<type>` 是数组，每个条目可以包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | Provider 稳定 ID。缺少时自动生成 `<type>-<序号>` 并在发布时持久化。 |
| `provider` | string | Provider 标识，通常与服务类型相同。 |
| `enabled` | boolean | 是否参与模型路由。 |
| `models` | string[] | 聊天模型列表，会自动去空格、去重。 |
| `embedding_models` | string[] | Embedding 模型列表。 |
| `server_url` | string | 上游 HTTP(S) 或 WebSocket 地址。 |
| `credentials` | object | Provider 凭证。不同服务需要的字段不同。 |
| `credential_list` | object[] | 多组轮换凭证，复杂结构建议使用高级 JSON。 |
| `model_map` | object | Provider 内部模型别名映射。 |
| `model_redirect` | object | Provider 内部模型重定向。 |
| `limit` / `embedding_limit` | object | `qps`、`qpm`、`rpm`、`concurrency`、`timeout`，数值不能为负。 |
| `use_proxy` | boolean | 覆盖全局代理策略。 |
| `timeout` | number | 单次请求超时秒数。 |

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
