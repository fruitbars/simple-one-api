# ollama接入使用指南

> [!NOTE]
> Ollama 的模型名和接口能力随版本变化，请以 Ollama 官方文档为准；simple-one-api 的当前配置字段见[配置参考](./configuration-reference.md)。

## 参考文档

开发文档地址：[https://github.com/ollama/ollama/blob/main/docs/api.md#generate-a-chat-completion](https://github.com/ollama/ollama/blob/main/docs/api.md#generate-a-chat-completion)
参考的是`Generate a chat completion`该部分描述来实现。

## 在simple-one-api中配置接入ollama
我们新建一个`ollama`的service，然后填入相关的配置信息
```json
{
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
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
