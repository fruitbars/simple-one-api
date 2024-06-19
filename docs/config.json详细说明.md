



# config.json详解

### 顶层字段说明

| 字段名           | 类型   | 说明                                                         |
| ---------------- | ------ | ------------------------------------------------------------ |
| `debug`          | 布尔值 | 是否开启debug模式（gin的debug模式），默认为false             |
| `log_level`      | 字符串 | 支持生产环境`prod`  开发环境：`dev`，dev日志非常详细         |
| `server_port`    | 字符串 | 服务地址，例如：":9090"                                      |
| `api_key`        | 字符串 | 客户端需要传入的api_key，例如："sk-123456"                   |
| `load_balancing` | 字符串 | 负载均衡策略，示例值："first"和"random"。first是取一个enabled，random是随机取一个enabled |
| `services`       | 对象   | 包含多个服务配置，每个服务对应一个大模型平台。               |
| `proxy`          | 对象   | 包含http_proxyh和https_proxy                                 |

### `services.<service>` 对象数组字段说明

每个服务包含一个或多个配置项。

| 字段名           | 类型       | 说明                                     |
| ---------------- | ---------- | ---------------------------------------- |
| `models`         | 字符串数组 | 支持的模型列表。                         |
| `enabled`        | 布尔值     | 是否启用该配置。                         |
| `credentials`    | 对象       | 凭证信息，根据不同服务可能包含不同字段。 |
| `model_map`      | 对象       | 支持模型设置别名。                       |
| `server_url`     | 字符串     | 服务器 URL，有些服务需要此字段。         |
| `model_redirect` | 对象       | 客户端传入的模型，进行重定向             |

### `credentials` 对象字段说明

根据不同服务，凭证信息包含不同的字段。

| 服务     | 字段名       | 类型   | 说明       |
| -------- | ------------ | ------ | ---------- |
| 讯飞星火 | `appid`      | 字符串 | 应用 ID。  |
|          | `api_key`    | 字符串 | API 密钥。 |
|          | `api_secret` | 字符串 | API 秘密。 |
| 百度千帆 | `api_key`    | 字符串 | API 密钥。 |
|          | `secret_key` | 字符串 | 秘密密钥。 |
| 腾讯混元 | `secret_id`  | 字符串 | 秘密 ID。  |
|          | `secret_key` | 字符串 | 秘密密钥。 |
| OpenAI   | `api_key`    | 字符串 | API 密钥。 |
| MiniMax  | `group_id`   | 字符串 | 组 ID。    |
|          | `api_key`    | 字符串 | API 密钥。 |



各个厂商详细的配置说明：https://github.com/fruitbars/simple-one-api/tree/main/docs

各个厂商详细的示例config：https://github.com/fruitbars/simple-one-api/tree/main/samples