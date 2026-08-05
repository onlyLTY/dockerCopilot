# 使用 Alpine 作为基础镜像（健康检查依赖 wget，tzdata 用于 TZ）
FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

ARG TARGETPLATFORM

COPY build/docker/dist/$TARGETPLATFORM/dockerCopilot ./dockerCopilot
COPY build/docker/etc/. ./etc
COPY build/docker/web/. ./web
COPY backend/start.sh ./start.sh

RUN apk add --no-cache tzdata wget \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

# secretKey 必须在运行时通过环境变量注入（compose 已强制要求）
ENV secretKey="" \
    WORKDIR="/app" \
    DOCKER_HOST="unix:///var/run/docker.sock" \
    BACKUP_DIR="/data/backups" \
    TZ="Asia/Shanghai"

VOLUME ["/data", "/compose", "/app/logs"]

EXPOSE 12712
CMD ["./start.sh"]