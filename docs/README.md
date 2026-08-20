# 文档目录

## 使用与配置

- [项目总览](../README.md)
- [配置参考](./configuration-reference.md)
- [桌面端开发](../cmd/desktop/README.md)
- [样例配置](../samples/)

## 客户端协议

- OpenAI Chat Completions：`POST /v1/chat/completions`
- OpenAI Responses / Codex：`POST /v1/responses`
- Anthropic Messages / Claude Code：`POST /v1/messages`
- 字段、鉴权、流式行为和限制见[配置参考](./configuration-reference.md#客户端协议)。

## 部署与维护

- [构建与发布](./build-and-release.md)
- [systemd 启动](./startup/systemd_startup.md)
- [nohup 启动](./startup/nohup_startup.md)

## 设计与版本

- [架构与交付状态](./architecture-v1.md)
- [更新日志](./CHANGELOG.md)

## Provider 接入指南

以下资料用于辅助找到厂商配置入口，不承诺额度、模型、价格、URL 或控制台截图仍然最新。配置结构以[配置参考](./configuration-reference.md)为准，供应商信息以其官方文档为准。

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

## 第三方客户端接入

- [沉浸式翻译](./在沉浸式翻译中使用simple-one-api.md)
