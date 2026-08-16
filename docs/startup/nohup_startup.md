# nohup 启动

脚本默认读取脚本目录下的 `config.json`：

```bash
chmod +x nohup_manage_simple_one_api.sh
./nohup_manage_simple_one_api.sh start
./nohup_manage_simple_one_api.sh stop
./nohup_manage_simple_one_api.sh restart
```

也可以在启动时传入配置文件：

```bash
./nohup_manage_simple_one_api.sh start /opt/simple-one-api/config.json
```

日志默认写入脚本目录下的 `simple-one-api.log`。
