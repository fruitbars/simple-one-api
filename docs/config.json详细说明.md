



# config.json详解

### 顶层字段说明

| 字段名              | 类型  | 说明                                                               |
|------------------|-----|------------------------------------------------------------------|
| `debug`          | 布尔值 | 是否开启debug模式（gin的debug模式），默认为false                                |
| `log_level`      | 字符串 | 支持生产环境`prod`  开发环境：`dev`，dev日志非常详细                               |
| `server_port`    | 字符串 | 服务地址，例如：":9090"                                                  |
| `api_key`        | 字符串 | 客户端需要传入的api_key，例如："sk-123456"                                   |
| `load_balancing` | 字符串 | 负载均衡策略，示例值："first"和"random"。first是取一个enabled，random是随机取一个enabled |
| `services`       | 对象  | 包含多个服务配置，每个服务对应一个大模型平台。                                          |
| `proxy`          | 对象  | 包含http_proxyh和https_proxy                                        |

### `services.<service>` 对象数组字段说明

每个服务包含一个或多个配置项。

| 字段名              | 类型    | 说明                   |
|------------------|-------|----------------------|
| `models`         | 字符串数组 | 支持的模型列表。             |
| `enabled`        | 布尔值   | 是否启用该配置。             |
| `credentials`    | 对象    | 凭证信息，根据不同服务可能包含不同字段。 |
| `model_map`      | 对象    | 支持模型设置别名。            |
| `server_url`     | 字符串   | 服务器 URL，有些服务需要此字段。   |
| `model_redirect` | 对象    | 客户端传入的模型，进行重定向       |

### `credentials` 对象字段说明

根据不同服务，凭证信息包含不同的字段。

| 服务      | 字段名          | 类型  | 说明      |
|---------|--------------|-----|---------|
| 讯飞星火    | `appid`      | 字符串 | 应用 ID。  |
|         | `api_key`    | 字符串 | API 密钥。 |
|         | `api_secret` | 字符串 | API 秘密。 |
| 百度千帆    | `api_key`    | 字符串 | API 密钥。 |
|         | `secret_key` | 字符串 | 秘密密钥。   |
| 腾讯混元    | `secret_id`  | 字符串 | 秘密 ID。  |
|         | `secret_key` | 字符串 | 秘密密钥。   |
| OpenAI  | `api_key`    | 字符串 | API 密钥。 |
| MiniMax | `group_id`   | 字符串 | 组 ID。   |
|         | `api_key`    | 字符串 | API 密钥。 |



各个厂商详细的配置说明：https://github.com/fruitbars/simple-one-api/tree/main/docs

各个厂商详细的示例config：https://github.com/fruitbars/simple-one-api/tree/main/samples



## 支持配置多个模型，可以随机负载均衡

客户端可以传入model名称为random，从而后台会随机找一个可用的模型进行调用。

```json
{
  "server_port":":9090",
  "load_balancing": "random",
  "services": {
    "qianfan": [
      {
        "models": ["yi_34b_chat", "ERNIE-Speed-8K", "ERNIE-Speed-128K", "ERNIE-Lite-8K", "ERNIE-Lite-8K-0922", "ERNIE-Tiny-8K"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx",
          "secret_key": "xxx"
        }
      }
    ],
    "xinghuo": [
      {
        "models": ["spark-lite"],
        "enabled": true,
        "credentials": {
          "appid": "xxx",
          "api_key": "xxx",
          "api_secret": "xxx"
        },
        "server_url": "ws://spark-api.xf-yun.com/v1.1/chat"
      }
    ],
    "hunyuan": [
      {
        "models": ["hunyuan-lite"],
        "enabled": true,
        "credentials": {
          "secret_id": "xxx",
          "secret_key": "xxx"
        }
      }
    ],
    "openai": [
      {
        "models": ["deepseek-chat"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.deepseek.com/v1"
      }
    ],
    "minimax": [
      {
        "models": ["abab6-chat"],
        "enabled": true,
        "credentials": {
          "group_id": "xxx",
          "api_key": "xxx"
        },
        "server_url": "https://api.minimax.chat/v1/text/chatcompletion_pro"
      }
    ]
  }
}
```



## 支持一个模型可配置多个`api_key`，并且可以随机负载均衡

这可以如果有多个credentials信息，可以新增一个数组项，进行输入

```json
{
    "api_key":"123456",
    "load_balancing": "random",
    "services": {
       "xinghuo": [
         {
           "models": ["spark-lite"],
           "enabled": true,
           "credentials": {
             "appid": "xxx",
             "api_key": "xxx",
             "api_secret": "xxx"
           }
         },
         {
           "models": ["spark-lite"],
           "enabled": true,
           "credentials": {
             "appid": "xxx",
             "api_key": "xxx",
             "api_secret": "xxx"
           }
         }
       ]
   }
}
```

## 支持设置一个对外总`api_key`

可以通过`api_key`字段来设置

```json
{
    "api_key":"123456",
    "load_balancing": "random",
    "services": {
       "xinghuo": [
         {
           "models": ["spark-lite"],
           "enabled": true,
           "credentials": {
             "appid": "xxx",
             "api_key": "xxx",
             "api_secret": "xxx"
           }
         }
       ]
   }
}
```



## 支持`random`模型，后台自动寻找配置的可用的模型

客户端可以传入model名称为random，从而后台会随机找一个可用的模型进行调用。

## 支持模型设置别名

model_redirect参数可以进行支持，意思是对客户端的model参数重定向到所支持的模型名称

```json
{
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": ["deepseek-ai/DeepSeek-Coder-V2-Instruct",
          "deepseek-ai/deepseek-v2-chat",
          "deepseek-ai/deepseek-llm-67b-chat",
          "alibaba/Qwen2-72B-Instruct",
          "alibaba/Qwen2-57B-A14B-Instruct",
          "alibaba/Qwen2-7B-Instruct",
          "alibaba/Qwen2-1.5B-Instruct",
          "alibaba/Qwen1.5-110B-Chat",
          "alibaba/Qwen1.5-32B-Chat",
          "alibaba/Qwen1.5-14B-Chat",
          "alibaba/Qwen1.5-7B-Chat",
          "01-ai/Yi-1.5-6B-Chat",
          "01-ai/Yi-1.5-9B-Chat",
          "01-ai/Yi-1.5-34B-Chat",
          "zhipuai/chatglm3-6B",
          "zhipuai/glm4-9B-chat"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "model_redirect": {
          "deepseek-v2-chat": "deepseek-ai/deepseek-v2-chat",
          "Qwen2-72B-Instruct": "alibaba/Qwen2-72B-Instruct"
        },
        "server_url":"https://api.siliconflow.cn/v1/chat/completions"
      }

    ]
  }
}
```



## 支持每一种模型服务设置服务的地址

我们看到支持openai协议的服务、minimax的服务都通过server_url设置了对应的服务地址

```json
{
  "server_port":":9090",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": ["deepseek-chat"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.deepseek.com/v1"
      }
    ],
    "minimax": [
      {
        "models": ["abab6-chat"],
        "enabled": true,
        "credentials": {
          "group_id": "xxx",
          "api_key": "xxx"
        },
        "server_url": "https://api.minimax.chat/v1/text/chatcompletion_pro"
      }
    ]
  }
}
```



## 支持全局代理模式
支持 http 或 socks5代理，参考文档《[simple‐one‐api代理配置说明](https://github.com/fruitbars/simple-one-api/wiki/simple%E2%80%90one%E2%80%90api%E4%BB%A3%E7%90%86%E9%85%8D%E7%BD%AE%E8%AF%B4%E6%98%8E)》



## 支持每个service设置qps或qpm或者concurrency

支持limit设置：qps - 每秒请求数、qpm（或rpm）- 每分钟请求出，concurrency-并发限制，timeout是限制情况下超时时间

```json
{
  "server_port": ":9090",
  "debug": false,
  "load_balancing": "random",
  "services": {
    "xinghuo": [
      {
        "models": ["spark-lite"],
        "enabled": true,
        "credentials": {
          "appid": "xxx",
          "api_key": "xxx",
          "api_secret": "xxx"
        },
        "limit": {
          "qps":2,
          "timeout": 10
        }
      }
    ]
  }
}
```

