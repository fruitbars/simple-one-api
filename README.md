## 功能

免费列表


## 文本生成
支持多种大模型：
    + [x] [OpenAI ChatGPT 系列模型](https://platform.openai.com/docs/guides/gpt/chat-completions-api)
    + [x] [百度智能云千帆大模型平台](https://qianfan.cloud.baidu.com/)
    + [x] [讯飞星火大模型](https://xinghuo.xfyun.cn/sparkapi)
    + [x] [腾讯混元大模型](https://cloud.tencent.com/product/hunyuan)
   
## 配置文件说明
我们以讯飞星火大模型为例子：
`services`中配置一项`xinghuo`
`models`可以配置：`["spark-lite","spark-v2.0","spark-pro","spark3.5-max"]`
`credentials`就是星火大模型的`appid`、`api_key`、`api_secret`

我们以免费的`spark-lite为例：
```json
{
  "load_balancing": "first",
  "services": {
     "xinghuo": [
        {
           "models": ["spark-lite"],
           "enabled": false,
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

如果需要支持更多的模型


参考完整配置示例
```json
{
  "load_balancing": "first",
  "services": {
    "qianfan": [
      {
        "models": ["yi_34b_chat","ERNIE-Speed-8K","ERNIE-Speed-128K","ERNIE-Lite-8K","ERNIE-Lite-8K-0922","ERNIE-Tiny-8K"],
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
        "enabled": false,
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
        "enabled": false,
        "credentials": {
          "secret_id": "xxx",
          "secret_key": "xxx"
        }
      }
    ],
    "openai": [
      {
        "models": ["deepseek-chat"],
        "enabled": false,
        "credentials": {
          "api_key": "xxx"
        },
        "server_url":"https://api.deepseek.com/v1"
      }
    ],
    "minimax": [
      {
        "models": ["abab6-chat"],
        "enabled": false,
        "credentials": {
          "group_id": "xxx",
          "api_key": "xxx"
        },
        "server_url":"https://api.minimax.chat/v1/text/chatcompletion_pro"
      }
    ]
  }
}

```