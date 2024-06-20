@echo off

REM 设置二进制文件的输出名称
SET BINARY_NAME=simple-one-api.exe

REM 编译项目
echo Building %BINARY_NAME%...
SET CGO_ENABLED=0
go build -o %BINARY_NAME%

echo Build completed.