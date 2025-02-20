# 基于官方的Go镜像作为基础镜像
FROM golang:1.23-alpine

# 设置Go模块代理
ENV GOPROXY=https://goproxy.cn

# 设置工作目录
WORKDIR /app

# 将当前目录下的所有文件复制到容器的工作目录中
COPY . .

# 下载依赖
RUN go mod download

# 编译Go项目
RUN go build -o im-service .

# 暴露服务端口
EXPOSE 8080

# 运行编译后的可执行文件
CMD ["./im-service"]