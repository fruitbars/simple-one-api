@echo off

REM 设置二进制文件的输出名称
SET BINARY_NAME=simple-one-api.exe

REM 转到源代码所在目录
cd cmd\simple-one-api

REM 编译项目
echo Building %BINARY_NAME%...
SET CGO_ENABLED=0
go build -o %BINARY_NAME%

echo Build completed. Copying the executable to the project root directory...

REM 拷贝编译后的文件到脚本当前目录
copy %BINARY_NAME% ..\..

REM 返回到原始目录
cd ..\..

echo Build and copy completed successfully!