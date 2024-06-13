# 通义千问DashScope申请使用流程

通义API是有DashScope提供的，本身做了openai的接口兼容：
[https://help.aliyun.com/zh/dashscope/developer-reference/compatibility-of-openai-with-dashscope](https://help.aliyun.com/zh/dashscope/developer-reference/compatibility-of-openai-with-dashscope)

因此`simple-one-api`直接是支持的，使用流程如下：

1. `api_key`获取说明文档：[https://help.aliyun.com/zh/dashscope/developer-reference/activate-dashscope-and-create-an-api-key](https://help.aliyun.com/zh/dashscope/developer-reference/activate-dashscope-and-create-an-api-key)

2. 在`simple-one-api`中可以这样配置：
```json
{
  "server_port": ":9099",
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": ["qwen-plus"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
      }
    ]
  }
}

```


