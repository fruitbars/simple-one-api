# Llama Family接入指南

文档地址：[https://llama.family/docs/api](https://llama.family/docs/api)

密钥管理：[https://llama.family/docs/secret](https://llama.family/docs/secret)

目前Llama Family提供免费的调用次数，限制信息如下：
> 速率限制：
> 
> 1.每天 8-22 点：接口限速每分钟 20 次并发
> 
> 2.每天 22-次日 8 点：接口限速每分钟 50 次并发

首先我们到Llama Family官网注册，并且到密钥管理后台获取到密钥。
![llama family](asset/llama family.jpg)

## 在simple-one-api中使用
`Llama Family`兼容`openai`协议，因此只需要在`services`中的`openai`项中加入相关配置即可。配置好密钥`api_key`以及服务地址`server_url`
```json
{
  "server_port": ":9099",
  "debug": false,
  "load_balancing": "random",
  "services": {
    "openai": [
      {
        "models": ["Atom-13B-Chat","Atom-7B-Chat","Atom-1B-Chat","Llama3-Chinese-8B-Instruct"],
        "enabled": true,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://api.atomecho.cn/v1"
      }
    ]
  }
}

```