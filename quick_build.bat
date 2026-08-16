@echo off
setlocal enabledelayedexpansion

cd /d "%~dp0"

if "%~1"=="" (
  for /f "delims=" %%G in ('go env GOOS') do set "TARGET_GOOS=%%G"
) else (
  set "TARGET_GOOS=%~1"
)
if "%~2"=="" (
  for /f "delims=" %%G in ('go env GOARCH') do set "TARGET_GOARCH=%%G"
) else (
  set "TARGET_GOARCH=%~2"
)

pnpm --dir web install --frozen-lockfile
if errorlevel 1 exit /b 1
pnpm --dir web build
if errorlevel 1 exit /b 1

REM 设置二进制文件的输出名称
SET BINARY_NAME=simple-one-api
if /i "%TARGET_GOOS%"=="windows" SET BINARY_NAME=simple-one-api.exe

REM 编译项目
echo Building %BINARY_NAME% for %TARGET_GOOS%/%TARGET_GOARCH%...
SET CGO_ENABLED=0
SET GOOS=%TARGET_GOOS%
SET GOARCH=%TARGET_GOARCH%
go build -o "%BINARY_NAME%"

echo Build completed.
