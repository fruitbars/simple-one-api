# simple-one-api:通过标准的 OpenAI API 格式访问的各种国产大模型，开箱即用

## 简介

目前市面上免费的使用国产的免费大模型越来越多，one-api对于个人用起来还是有点麻烦，就想要一个不要统计、流量、计费等等的适配程序即可。

还有一点是：即使有些厂商说兼容openai的接口，但是实际上还是存在些许差异的！！！

**simple-one-api**主要是解决以上2点，旨在兼容多种大模型接口，并统一对外提供 OpenAI 接口。通过该项目，用户可以方便地集成和调用多种大模型，简化了不同平台接口差异带来的复杂性。



### 免费大模型列表

| 大模型             | 免费版本                                                     | 控制台（api_key等）                                          | 文档地址                                               | 备注                                                         |
| ------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------------ |
| 讯飞星火大模型     | `spark-lite`                                                 | [链接](https://console.xfyun.cn/services/cbm)                | [链接](https://www.xfyun.cn/doc/spark/Web.html)        | tokens：总量无限<br>QPS：2<br>有效期：不限                   |
| 百度千帆大模型平台 | `yi_34b_chat`, `ERNIE-Speed-8K`, `ERNIE-Speed-128K`, `ERNIE-Lite-8K`, `ERNIE-Lite-8K-0922`, `ERNIE-Tiny-8K` | [链接](https://console.bce.baidu.com/qianfan/ais/console/applicationConsole/application) | [链接](https://cloud.baidu.com/doc/WENXINWORKSHOP/s/klqx7b1xf) | Lite、Speed-8K：RPM = 300，TPM = 300000<br>Speed-128K：RPM = 60，TPM = 300000 |
| 腾讯混元大模型     | `hunyuan-lite`                                               | [链接](https://console.cloud.tencent.com/cam/capi)           | [链接](https://cloud.tencent.com/document/api/1729/105701) | 限制并发数为 5 路                                            |

#### 备注信息
- **讯飞星火大模型**:
   - **tokens**: 总量无限
   - **QPS**: 2
   - **有效期**: 不限
   - **文档地址**：[https://www.xfyun.cn/doc/spark/Web.html](https://www.xfyun.cn/doc/spark/Web.html)
- **百度千帆大模型平台**:
   - **Lite、Speed-8K**: RPM = 300，TPM = 300000
   - **Speed-128K**: RPM = 60，TPM = 300000]
   - **文档地址**：[https://cloud.baidu.com/doc/WENXINWORKSHOP/s/klqx7b1xf](https://cloud.baidu.com/doc/WENXINWORKSHOP/s/klqx7b1xf)
- **腾讯混元大模型**:
   - **限制并发数**: 5 路
   - **文档地址**：[https://cloud.tencent.com/document/api/1729/105701](https://cloud.tencent.com/document/api/1729/105701)

## 功能

### 文本生成

支持多种大模型：

- [x] [百度智能云千帆大模型平台](https://qianfan.cloud.baidu.com/)
- [x] [讯飞星火大模型](https://xinghuo.xfyun.cn/sparkapi)
- [x] [腾讯混元大模型](https://cloud.tencent.com/product/hunyuan)
- [x] [OpenAI ChatGPT 系列模型](https://platform.openai.com/docs/guides/gpt/chat-completions-api)
    - [x] [Deep-Seek](https://platform.deepseek.com/api-docs/zh-cn/)
- [x] [MiniMax](https://platform.minimaxi.com/document/guides/chat-model/pro)

如果兼容OpenAI的接口，那么直接就可以使用了。

## 安装

### 源码安装

1. 克隆本仓库：

   ```bash
   git clone https://github.com/fruitbars/simple-one-api.git
   ```

2. 编译程序：

   ``````shell
   chmod +x build_windows.sh
   chmod +x build_linux.sh
   chmod +x build_macos.sh
   ``````

   对于不同的平台可以执行不同的脚本

   ```bash
   ./build_windows.sh           # 普通编译
   ./build_windows.sh --rebuild # 全量重新编译
   
   ./build_linux.sh           # 普通编译
   ./build_linux.sh --rebuild # 全量重新编译
   
   ./build_macos.sh           # 普通编译
   ./build_macos.sh --rebuild # 全量重新编译
   ```

3. 配置你的模型服务和凭证：

   在 `config.json` 文件中添加你的模型服务和凭证信息，参考下文的配置文件说明。

### 直接下载

[前往Releases页面](https://github.com/fruitbars/simple-one-api/releases)

## 使用方法

1. 启动服务：

   ```bash
   ./simple-one-api [config](可选项，默认为config.json)
   ```

2. 调用 API：

   现在，你可以通过 OpenAI 兼容的接口调用你配置的各大模型服务。服务地址: `http://host:port/v1`,`api-key`可以任意设置

   支持模型名称设置为`random`，后台会自动找一个`"enabled": true`的模型来使用。

## 配置文件示例


```json
{
    "load_balancing": "first",
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
```

## 配置文件说明

配置文件采用 JSON 格式，以下是各字段的详细说明。

### 顶层字段说明

| 字段名              | 类型   | 说明                                                               |
|------------------| ------ |------------------------------------------------------------------|
| `server_port`    | 字符串 | 服务地址，例如：":9090"                                                  |
| `api_key`        | 字符串 | 客户端需要传入的api_key，例如："sk-123456"                                   |
| `load_balancing` | 字符串 | 负载均衡策略，示例值："first"和"random"。first是取一个enabled，random是随机取一个enabled |
| `services`       | 对象   | 包含多个服务配置，每个服务对应一个大模型平台。                                          |

### `services.<service>` 对象数组字段说明

每个服务包含一个或多个配置项。

| 字段名        | 类型       | 说明                                     |
| ------------- | ---------- | ---------------------------------------- |
| `models`      | 字符串数组 | 支持的模型列表。                         |
| `enabled`     | 布尔值     | 是否启用该配置。                         |
| `credentials` | 对象       | 凭证信息，根据不同服务可能包含不同字段。 |
| `server_url`  | 字符串     | 服务器 URL，有些服务需要此字段。         |

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

### 示例配置文件

以下是一个完整的配置示例，涵盖了多个大模型平台和不同模型：

```json
{
  "server_port":":9090",
  "load_balancing": "first",
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
## FAQ
### 如何设置一个对外的apikey？
可以通过`api_key`字段来设置
```json
{
    "api_key":"123456",
    "load_balancing": "random",
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
```
### 单个模型如何配置多个credentials自动负载？
 以客户端选择spark-lite为例，可以按照下面这样配置，会随机credentials

```json
{
    "api_key":"123456",
    "load_balancing": "random",
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
```

### 如何让后台模型随机使用？
`load_balancing`就是为自动选择模型来配置的，支持`random`，自动随机选一个`enabled`为`true`的模型

```json
{
    "api_key":"123456",
    "load_balancing": "random",
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
```
## 贡献

我们欢迎任何形式的贡献。如果你有任何建议或发现了问题，请通过提交 issue 或 pull request 的方式与我们联系。