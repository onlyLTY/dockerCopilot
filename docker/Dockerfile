# 使用 Alpine 作为基础镜像
FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

ARG TARGETPLATFORM
# 复制适当的平台二进制文件
COPY dist/$TARGETPLATFORM/dockerCopilot /app/dockerCopilot

# 复制其他需要的文件或目录，例如 start.sh
COPY . ${WORKDIR}
RUN apk add --no-cache tzdata \
    && chmod +x start.sh

# 设置环境变量
ENV secretKey="" \
    DOCKER_HOST="tcp://127.0.0.1:9375" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]
# 设置启动命令
CMD ["./start.sh"]