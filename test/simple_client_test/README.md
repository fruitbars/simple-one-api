# 手工 SimpleClient 冒烟测试

这个目录不是自动化单元测试；`main.go` 会实际调用配置中的 Provider，依次验证流式和非流式请求。

从仓库根目录运行，并显式传入配置文件：

```sh
go run ./test/simple_client_test ./config.json
```

不传参数时默认读取当前工作目录的 `config.json`。配置可能包含真实上游密钥，请不要提交本地配置或运行日志。
