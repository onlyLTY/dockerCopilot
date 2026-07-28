# 使用 Alpine 作为基础镜像
FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

ARG TARGETPLATFORM

COPY build/docker/dist/$TARGETPLATFORM/dockerCopilot ./dockerCopilot
COPY build/docker/etc/. ./etc
COPY build/docker/web/. ./web
COPY backend/start.sh ./start.sh

RUN apk add --no-cache tzdata \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

# 设置环境变量
ENV secretKey="" \
    WORKDIR="/app" \
    DOCKER_HOST="unix:///var/run/docker.sock" \
    BACKUP_DIR="/data/backups" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]

EXPOSE 12712
# 设置启动命令
CMD ["./start.sh"]