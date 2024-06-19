<p align="right">
   <strong>English</strong> | <a href="./README.md">中文</a> 
</p>

# simple-one-api: Various large models accessible through a standardized OpenAI API format, ready to use out of the box

## Introduction

There are an increasing number of free large-scale models available on the market, and one-api can be somewhat cumbersome for personal use. What's desired is an adaptation program that does not require accounting, traffic, billing, etc.

Another point is that even though some manufacturers claim compatibility with the openai interface, there are still some differences in reality!!!

**simple-one-api** mainly addresses the above two points, aiming to be compatible with various large model interfaces and uniformly providing the OpenAI interface. Through this project, users can easily integrate and call various large models, simplifying the complexity brought by different platform interface differences.

### Free Large Model List

| Large Model             | Free Version                                                         | Free Limitations                                                | Console (api_key etc.)                                        | Documentation URL                                             |
|-------------------------|----------------------------------------------------------------------|-----------------------------------------------------------------|--------------------------------------------------------------|----------------------------------------------------------------|
| Cloudflare Workers AI   | `All Models`                                                         | Free to use 10,000 times per day, 300,000 times per month; unlimited in test version | [Access Link](https://dash.cloudflare.com/)                  | [Documentation View](https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/) |
| ByteDance Coze.com      | Various Models including Function call, General question-asking models and more | Current Coze API free for developers, with API request limit per space: QPS (requests per second): 2<br/>QPM (requests per minute): 60<br/>QPD (requests per day): 3000 | [Access Link](https://www.coze.cn/space)                     | [Documentation View](https://www.coze.cn/docs/developer_guides/coze_api_overview) |
| Llama Family            | Various Models including Chat models with different capabilities    | 1. 8 AM to 10 PM: API rate limit of 20 requests per minute<br/>2. 10 PM to 8 AM next day: API rate limit of 50 requests per minute | [Access Link](https://llama.family/docs/secret)              | [Documentation View](https://llama.family/docs/chat-completion-v1) |
| Groq                    | Various Models including different configurations of AI models      | rpm is 30, rpd is 14400, TOKENS PER MINUTE also limited        | [Access Link](https://console.groq.com/keys)                 | [Documentation View](https://console.groq.com/docs/text-chat) |

#### Notes

- **Cloudflare Workers AI**
  - **Limitations**: Free to use 10,000 times per day, 300,000 times per month; unlimited in test version
  - **Documentation URL**: [https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/](https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/)
  - **Application Process**: [docs/Cloudflare_Workers_AI Application Process.md](docs/Cloudflare_Workers_AI Application Process.md)
- **ByteDance Coze.com**
   - **Limitations**: QPS: 2, QPM: 60, QPD: 3000
   - **Documentation URL**: https://www.coze.com/docs/developer_guides/coze_api_overview
   - **Application Process**: [docs/coze.cn API Application Process.md](docs/coze.cn API Application Process.md)
- **Groq**
   - **Limitations**: rpm is 30, rpd is 14400, TOKENS PER MINUTE also limited
   - **Documentation URL**: https://console.groq.com/docs/text-chat
   - **Application Process**: [docs/Groq Integration Guide.md](docs/Groq Integration Guide.md)

## Features

### Text Generation

Support for multiple large models:
- [x] OpenAI ChatGPT series models
    - [x] [OpenAI](https://platform.openai.com/docs/guides/gpt/chat-completions-api)
    - [x] [Cloudflare Workers AI](https://developers.cloudflare.com/workers-ai/configuration/open-ai-compatibility/)
    - [x] [Azure OpenAI](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference)
    - [x] [Groq](https://console.groq.com/docs/text-chat)

- [x] ByteDance Coze
    - [x] [Coze.com](https://www.coze.com/docs/developer_guides/coze_api_overview)

- [x] [Ollama](https://github.com/ollama/ollama/blob/main/docs/api.md)

If compatible with the OpenAI interface, it can be used directly. See the document [docs/Compatibility with OpenAI Model Protocol

 Integration Guide.md](docs/Compatibility with OpenAI Model Protocol Integration Guide.md)

### Supported Features
- Support for configuring multiple models, can balance load randomly
- Support for configuring multiple `api_key` for a model, and can balance load randomly
- Support for setting a global `api_key`
- Support for `random` model, automatically finds a configured available model
- Support for setting aliases for models
- Support for setting the service address for each model service
- Compatible with OpenAI's interface, supports both /v1 and /v1/chat/completions paths
- For models not supporting 'system', simple-one-api will include it in the first prompt for uniformity (e.g., in immersive translation, models not supporting 'system' can also be called normally)
- Support for global proxy mode
- Support for setting qps or qpm or concurrency for each service

### Update Log

View [CHANGELOG.md](docs/CHANGELOG.md) for detailed update history of this project.

## Installation

### Source Installation

1. Clone this repository:

```bash
git clone https://github.com/fruitbars/simple-one-api.git
```

#### Quick Compilation and Usage

First, ensure you have installed Go, version should be 1.18 or above, refer to the official tutorial for installation: [https://go.dev/doc/install](https://go.dev/doc/install)
You can check the Go version with `go version`.

**linux/macOS**

```shell
chmod +x quick_build.sh
./quick_build.sh
```

This will generate `simple-one-api` in the current directory.

**Windows**
Double-click `quick_build.bat` to execute.

```bat
quick_build.bat
```

This will generate `simple-one-api.exe` in the current directory.

**Cross-compile for different platforms**

Sometimes you need to compile versions for different platforms, such as windows, linux, macOS; after installing Go, execute `build.sh`

 ```shell
chmod +x build.sh
./build.sh
 ```

This will automatically compile executable files for the above three platforms in different architectures, generated in the `build` directory.

**Next, configure your model services and credentials:**
Add your model service and credential information in the `config.json` file, refer to the configuration file description below.

### Direct Download

[Go to Releases Page](https://github.com/fruitbars/simple-one-api/releases)


## How to Use

### Direct Start
Default to read and start the `config.json` in the same directory as `simple-one-api`
   ```bash
   ./simple-one-api
   ```
If you want to specify the path of `config.json`, you can start like this
   ```bash
   ./simple-one-api /path/to/config.json
   ```

### Docker Start

Here are the steps to deploy `simple-one-api` using Docker:
**Running**
Run the Docker container using the following command while mounting your configuration file `config.json`:

```sh
docker run -d --name simple-one-api -p 9090:9090 -v /path/to/config.json:/app/config.json fruitbars/simple-one-api
```

**Note:** Make sure to replace /path/to/config.json with the absolute path of the config.json file on your host.

**View Container Logs**
You can view the log output of the container with the following command:

```sh
docker logs -f simple-one-api
```

or

```sh
docker logs -f <container_id>
```

Where <container_id> is the container ID, which can be viewed using the docker ps command.

#### Docker Compose Start Steps

1. **Configuration File**: In `docker-compose.yml`, first make sure you have replaced the path of your `config.json` file with the correct absolute path.

2. **Start Container**:
   Using Docker Compose to start the service, you can run the following command in the directory containing `docker-compose.yml`:

   ```sh
   docker-compose up -d
   ```

   This command will start the `simple-one-api` service in the background.

Other command references can be found in the docker-compose documentation.

### Other Start Methods
Other start methods:
- [nohup Start](docs/startup/nohup_startup.md)
- [systemd Start](docs/startup/systemd_startup.md)


### Calling the API

Now, you can call your configured large model services through the OpenAI compatible interface. Service address: `http://host:port/v1`, `api-key` can be set arbitrarily

Supported model names set to `random`, the backend will automatically find a model marked `"enabled": true` to use.

## Configuration File Example (Cloudflare Workers AI as example)

```json
{
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": [
          "@cf/meta/llama-

2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      }
    ]
  }
}

```

Other model's configuration file examples can be found at

## Configuration File Description

Refer to the document: [Detailed config.json Explanation](docs/config.json详细说明.md)

Detailed configuration descriptions for each vendor: [https://github.com/fruitbars/simple-one-api/tree/main/docs](https://github.com/fruitbars/simple-one-api/tree/main/docs)

Detailed example configs for each vendor: [https://github.com/fruitbars/simple-one-api/tree/main/samples](https://github.com/fruitbars/simple-one-api/tree/main/samples)

### More Complete Configuration File Example

Here is a complete configuration example, covering multiple large model platforms and different models:

```json
{
  "server_port":":9090",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": [
          "@cf/meta/llama-2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      },
      {
        "models": ["llama3-70b-8192","llama3-8b-8192","gemma-7b-it","mixtral-8x7b-32768"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://api.groq.com/openai/v1"
      }
    ],
    "cozecom": [
      {
        "models": ["xxx"],
        "enabled": true,
        "credentials": {
          "token": "xxx"
        },
        "server_url": "https://api.coze.com/open_api/v2/chat"
      }
    ],
    "azure": [
      {
        "models": ["gpt-4o"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://xxx.openai.azure.com/openai/deployments/xxx/completions?api-version=2024-05-13"
      }
    ],
    "ollama": [
      {
        "models": ["llama2"],
        "enabled": true,
        "server_url":"http://127.0.0.1:11434/api/chat"
      }
    ]
  }
}
```
## FAQ
### How to use in immersive translation?

Refer to [docs/How to Use simple-one-api in Immersive Translation](docs/How to Use simple-one-api in Immersive Translation.md)

### Is concurrency limiting supported?

Yes, it is supported. Refer to the following configuration, the free Coze.com model has a 2qps limit, so it can be set like this

```json
{
  "server_port": ":9090",
  "debug": false,
  "load_balancing": "random",
  "services": {
    "cozecom": [
      {
        "models": ["xxx"],
        "enabled": true,
        "credentials": {
          "token": "xxx"
        },
        "limit": {
          "qps":2,
          "timeout": 10
        },
        "server_url": "https://api.coze.com/open_api/v2/chat"
      }
    ]
  }
}
```

### How to set an external apikey?

It can be set through the `api_key` field
```json
{
  "qpi_key": "123456",
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": [
          "@cf/meta/llama-2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      }
    ]
  }
}
```
### How to configure multiple credentials for a single model to automatically load balance?
For client selection of spark-lite, you can configure it as follows, randomly choosing credentials

```json
{
  "server_port": ":9099",
 

 "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": [
          "@cf/meta/llama-2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      },
      {
        "models": [
          "@cf/meta/llama-2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      }
    ]
  }
}
```
### How to let the backend randomly select a model to use?
`load_balancing` is configured to automatically select a model, supporting `random`, automatically choosing a model with `enabled` set to `true`

```json
{
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": [
          "@cf/meta/llama-2-7b-chat-int8"
        ],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url": "https://api.cloudflare.com/client/v4/accounts/0b4a4013591101f6f5657fcb68f32043/ai/v1/chat/completions"
      }
    ],
    "ollama": [
      {
        "models": ["llama2"],
        "enabled": true,
        "server_url":"http://127.0.0.1:11434/api/chat"
      }
    ]
  }
}
```

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=fruitbars/simple-one-api&type=Date)](https://star-history.com/#fruitbars/simple-one-api&Date)

## Contribution

We welcome any form of contribution. If you have any suggestions or have found any issues, please contact us by submitting an issue or pull request.
```