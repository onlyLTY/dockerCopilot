# 使用官方Go镜像作为构建环境
FROM golang:1.20 AS build-env

# 设置工作目录
WORKDIR /app

# 添加源码到容器中
COPY . .

# 设置Go模块代理（可选，如果在国内可以使用这些代理加速模块下载）
# ENV GO111MODULE=on
# ENV GOPROXY=https://goproxy.cn,direct

# 编译Go应用程序
RUN go build -o onekeyupdate .

# 使用scratch或alpine作为基础镜像，这将使得最终的镜像尺寸更小
FROM alpine

# 从构建环境中拷贝编译好的应用程序
COPY --from=build-env /app/onekeyupdate /onekeyupdate

ENV account="" \
    secretKey="" \
    hubURL="https://hub.docker.com"

# 指定容器启动时运行的命令
ENTRYPOINT ["/onekeyupdate"]