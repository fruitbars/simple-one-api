# 使用一个轻量级的基础镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 复制二进制文件到容器中
COPY build/simple-one-api-linux/simple-one-api .

# 暴露应用运行的端口（根据你的应用设置，假设为8080）
EXPOSE 9090

# 运行可执行文件
CMD ["./simple-one-api"]
