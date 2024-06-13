# 使用 systemd 服务

您也可以使用我们提供的脚本 `install_simple_one_api_service.sh` 来设置服务。首先，您需要在脚本中指定应用的工作目录：
```bash
WORKING_DIRECTORY="/path/to/your/application"
```
接着，为脚本文件设置执行权限，并执行安装：
```bash
chmod +x install_simple_one_api_service.sh
./install_simple_one_api_service.sh
```
安装完成后，您可以通过以下 systemd 命令来管理服务：
- 启动服务：
```bash
sudo systemctl start simple-one-api
```
- 停止服务：
```bash
sudo systemctl stop simple-one-api
```
- 重启服务：
```bash
sudo systemctl restart simple-one-api
```