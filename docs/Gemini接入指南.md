# Gemini接入指南

> [!WARNING]
> 这是历史接入指南。模型、额度、控制台和认证方式可能已经变化，请先核对 Gemini 官方文档；simple-one-api 的当前配置字段以[配置参考](./configuration-reference.md)为准。

文档地址：https://ai.google.dev/gemini-api/docs/api-overview

后台地址：https://aistudio.google.com/app/apikey

## 在simple-one-api中使用

新建一个`gemini`,填写上相关配置即可。注意Gemini免费版限制：15RPM（每分钟请求数）;100万 TPM（每分钟令牌）;1500 RPD（每天请求数）
因此可以在`simple-one-api`中设置`limit`设置。

```json
{
  "server_port": ":9099",
  "log_level": "prodj",
  "load_balancing": "random",
  "services": {
    "gemini": [
      {
        "models": ["gemini-1.5-flash"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "limit": {
          "rpm": 15,
          "timeout":120
        }
      }
    ]
  }
}

```
