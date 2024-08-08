# 兼容OpenAI模型协议接入指南

## 兼容OpenAI协议模型汇总
目前兼容OpenAI协议的模型包括
- Cloudflare_Workers_AI
- DeepSeek
- 智谱glm
- 阿里DashScope
- 字节火山方舟 
- 零一万物
- groq

## 接入方式示例
官方OpenAI直接可以接入，对于其他厂商的模型，我们只需要配置好模型名称以及该模型的服务地址即`server_url`即可。
这里为了方便直接兼容两种形式的API地址，可以选择或者不带上`/chat/completions`，复制的时候比较方便！
```
 https://api.deepseek.com/v1
 https://api.deepseek.com/v1/chat/completions
```
以上两种形式地址都是可以的。

### DeepSeek接入simple-one-api
我们以DeepSeek模型为例，目前DeepSeek提供的模型是`deepseek-chat`,然后其服务地址是`https://api.deepseek.com/v1` 
因此配置信息如下：
```json
{
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
    ]
  }
}
```
DeepSeek接入可以参考文档[docs/deepseek模型申请使用流程.md](https://github.com/fruitbars/simple-one-api/blob/main/docs/deepseek%E6%A8%A1%E5%9E%8B%E7%94%B3%E8%AF%B7%E4%BD%BF%E7%94%A8%E6%B5%81%E7%A8%8B.md)

### 智谱glm接入simple-one-api

```json
{
  "services": {
    "openai": [
      {
        "models": ["glm-4","glm-3-turbo"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://open.bigmodel.cn/api/paas/v4/chat/completions"
      }
    ]
  }
}
```
智谱详细接入可以参考文档[docs/智谱glm模型申请使用流程.md](https://github.com/fruitbars/simple-one-api/blob/main/docs/%E6%99%BA%E8%B0%B1glm%E6%A8%A1%E5%9E%8B%E7%94%B3%E8%AF%B7%E4%BD%BF%E7%94%A8%E6%B5%81%E7%A8%8B.md)

### 零一万物接入simple-one-api
文档中心：https://platform.lingyiwanwu.com/docs
API 服务地址：https://api.lingyiwanwu.com/v1/chat/completions
Key管理：https://platform.lingyiwanwu.com/apikeys

```json
{
    "services": {
        "openai": [
            {
                "models": [
                    "yi-large",
                    "yi-spark",
                    "yi-medium",
                    "yi-medium-200k",
                    "yi-large-turbo"
                ],
                "enabled": true,
                "credentials": {
                    "api_key": "xxx"
                },
                "server_url": "https://api.lingyiwanwu.com/v1/chat/completions"
            }
        ]
    }
}
```

### Nvidia 
文档中心：https://docs.api.nvidia.com
API 服务地址：https://integrate.api.nvidia.com/v1/chat/completions
Key管理：https://build.nvidia.com/explore/discover

```json
{
    "services": {
        "openai": [
            {
                "models": [
                    "01-ai/yi-large",
                    "aisingapore/sea-lion-7b-instruct",
                    "bigcode/starcoder2-7b",
                    "bigcode/starcoder2-15b",
                    "databricks/dbrx-instruct",
                    "deepseek-ai/deepseek-coder-6.7b-instruct",
                    "google/gemma-7b",
                    "google/gemma-2b",
                    "google/gemma-2-9b-it",
                    "google/gemma-2-27b-it",
                    "google/codegemma-1.1-7b",
                    "google/codegemma-7b",
                    "google/recurrentgemma-2b",
                    "ibm/granite-34b-code-instruct",
                    "ibm/granite-8b-code-instruct",
                    "mediatek/breeze-7b-instruct",
                    "meta/codellama-70b",
                    "meta/llama2-70b",
                    "meta/llama3-8b",
                    "meta/llama3-70b",
                    "meta/llama-3.1-8b-instruct",
                    "meta/llama-3.1-70b-instruct",
                    "meta/llama-3.1-405b-instruct",
                    "microsoft/phi-3-medium-128k-instruct",
                    "microsoft/phi-3-medium-4k-instruct",
                    "microsoft/phi-3-mini-128k-instruct",
                    "microsoft/phi-3-mini-4k-instruct",
                    "microsoft/phi-3-small-128k-instruct",
                    "microsoft/phi-3-small-8k-instruct",
                    "mistralai/codestral-22b-instruct-v0.1",
                    "mistralai/mamba-codestral-7b-v0.1",
                    "mistralai/mistral-7b-instruct",
                    "mistralai/mistral-7b-instruct-v0.3",
                    "mistralai/mixtral-8x7b-instruct",
                    "mistralai/mixtral-8x22b-instruct",
                    "mistralai/mistral-large",
                    "nv-mistralai/mistral-nemo-12b-instruct",
                    "nvidia/llama3-chatqa-1.5-70b",
                    "nvidia/llama3-chatqa-1.5-8b",
                    "nvidia/nemotron-4-340b-instruct",
                    "nvidia/nemotron-4-340b-reward",
                    "nvidia/usdcode-llama3-70b-instruct",
                    "seallms/seallm-7b-v2.5",
                    "snowflake/arctic",
                    "upstage/solar-10.7b-instruct"
                ],
                "model_redirect": {
                    "yi-large": "01-ai/yi-large",
                    "sea-lion-7b-instruct": "aisingapore/sea-lion-7b-instruct",
                    "starcoder2-7b": "bigcode/starcoder2-7b",
                    "starcoder2-15b": "bigcode/starcoder2-15b",
                    "dbrx-instruct": "databricks/dbrx-instruct",
                    "deepseek-coder-6.7b-instruct": "deepseek-ai/deepseek-coder-6.7b-instruct",
                    "gemma-7b": "google/gemma-7b",
                    "gemma-2b": "google/gemma-2b",
                    "gemma-2-9b-it": "google/gemma-2-9b-it",
                    "gemma-2-27b-it": "google/gemma-2-27b-it",
                    "codegemma-1.1-7b": "google/codegemma-1.1-7b",
                    "codegemma-7b": "google/codegemma-7b",
                    "recurrentgemma-2b": "google/recurrentgemma-2b",
                    "granite-34b-code-instruct": "ibm/granite-34b-code-instruct",
                    "granite-8b-code-instruct": "ibm/granite-8b-code-instruct",
                    "breeze-7b-instruct": "mediatek/breeze-7b-instruct",
                    "codellama-70b": "meta/codellama-70b",
                    "llama2-70b": "meta/llama2-70b",
                    "llama3-8b": "meta/llama3-8b",
                    "llama3-70b": "meta/llama3-70b",
                    "llama-3.1-8b-instruct": "meta/llama-3.1-8b-instruct",
                    "llama-3.1-70b-instruct": "meta/llama-3.1-70b-instruct",
                    "llama-3.1-405b-instruct": "meta/llama-3.1-405b-instruct",
                    "phi-3-medium-128k-instruct": "microsoft/phi-3-medium-128k-instruct",
                    "phi-3-medium-4k-instruct": "microsoft/phi-3-medium-4k-instruct",
                    "phi-3-mini-128k-instruct": "microsoft/phi-3-mini-128k-instruct",
                    "phi-3-mini-4k-instruct": "microsoft/phi-3-mini-4k-instruct",
                    "phi-3-small-128k-instruct": "microsoft/phi-3-small-128k-instruct",
                    "phi-3-small-8k-instruct": "microsoft/phi-3-small-8k-instruct",
                    "codestral-22b-instruct-v0.1": "mistralai/codestral-22b-instruct-v0.1",
                    "mamba-codestral-7b-v0.1": "mistralai/mamba-codestral-7b-v0.1",
                    "mistral-7b-instruct": "mistralai/mistral-7b-instruct",
                    "mistral-7b-instruct-v0.3": "mistralai/mistral-7b-instruct-v0.3",
                    "mixtral-8x7b-instruct": "mistralai/mixtral-8x7b-instruct",
                    "mixtral-8x22b-instruct": "mistralai/mixtral-8x22b-instruct",
                    "mistral-large": "mistralai/mistral-large",
                    "mistral-nemo-12b-instruct": "nv-mistralai/mistral-nemo-12b-instruct",
                    "llama3-chatqa-1.5-70b": "nvidia/llama3-chatqa-1.5-70b",
                    "llama3-chatqa-1.5-8b": "nvidia/llama3-chatqa-1.5-8b",
                    "nemotron-4-340b-instruct": "nvidia/nemotron-4-340b-instruct",
                    "nemotron-4-340b-reward": "nvidia/nemotron-4-340b-reward",
                    "usdcode-llama3-70b-instruct": "nvidia/usdcode-llama3-70b-instruct",
                    "seallm-7b-v2.5": "seallms/seallm-7b-v2.5",
                    "arctic": "snowflake/arctic",
                    "solar-10.7b-instruct": "upstage/solar-10.7b-instruct"
                },
                "enabled": true,
                "credentials": {
                    "api_key": "nvapi--xxx"
                },
                "server_url": "https://integrate.api.nvidia.com/v1"
            }
        ]
    }
}
```

