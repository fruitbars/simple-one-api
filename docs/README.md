# 文档目录

## 先看这几篇

- [项目总览](../README.md)
- [配置参考](./configuration-reference.md)
- [构建与发布](./build-and-release.md)
- [v1 架构说明](./architecture-v1.md)
- [桌面端开发](../cmd/desktop/README.md)
- [更新日志](./CHANGELOG.md)

## 客户端协议

- OpenAI Chat Completions：`POST /v1/chat/completions`
- OpenAI Responses / Codex：`POST /v1/responses`
- Anthropic Messages / Claude Code：`POST /v1/messages`
- 字段、鉴权、流式行为和限制见[配置参考](./configuration-reference.md#客户端协议)。

## 部署

- [systemd 启动](./startup/systemd_startup.md)
- [nohup 启动](./startup/nohup_startup.md)

## Provider 接入指南（历史资料）

以下文档保留已有链接和接入背景，但不是当前额度、模型或控制台信息的权威来源。使用前请先核对厂商官方文档，配置结构以[配置参考](./configuration-reference.md)为准。

- [OpenAI 兼容协议](./兼容OpenAI模型协议接入指南.md)
- [Cloudflare Workers AI](./Cloudflare_Workers_AI申请使用流程.md)
- [DeepSeek](./deepseek模型申请使用流程.md)
- [Gemini](./Gemini接入指南.md)
- [Groq](./groq接入指南.md)
- [Ollama](./ollama接入指南.md)
- [百度千帆](./百度千帆speed和lite模型申请流程.md)
- [腾讯混元](./腾讯混元hunyuan-lite模型申请流程.md)
- [讯飞星火](./讯飞星火spark-lite模型申请流程.md)
- [通义千问 DashScope](./通义千问DashScope申请使用流程.md)
- [火山方舟](./火山方舟大模型接入指南.md)
- [智谱 GLM](./智谱glm模型申请使用流程.md)
- [零一万物](./零一万物接入指南.md)
- [Llama Family](./llama_family接入指南.md)

## 客户端接入（历史资料）

- [沉浸式翻译](./在沉浸式翻译中使用simple-one-api.md)
