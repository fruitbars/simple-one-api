# 使用 systemd 服务

使用仓库根目录的 `install_simple_one_api_service.sh` 安装。第一个参数是应用目录，第二个可选参数是配置文件路径：

```bash
chmod +x install_simple_one_api_service.sh
sudo ./install_simple_one_api_service.sh /opt/simple-one-api /opt/simple-one-api/config.json
```

脚本会把日志写入 systemd journal，并将 SQLite 路径设置为应用目录下的 `config.db`。

```bash
sudo systemctl start simple-one-api
sudo systemctl stop simple-one-api
sudo systemctl restart simple-one-api
sudo journalctl -u simple-one-api -f
```
