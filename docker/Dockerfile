# 使用 Alpine 作为基础镜像
FROM alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce
LABEL org.opencontainers.image.authors="autunn"
WORKDIR /app

ARG TARGETPLATFORM

COPY dc-back/dist/$TARGETPLATFORM/dockerCopilot ./dockerCopilot
COPY dc-back/etc/. ./etc
COPY start.sh ./start.sh

RUN apk add --no-cache tzdata \
    && chmod +x start.sh \
    && chmod +x dockerCopilot

# 设置环境变量
ENV DOCKER_HOST="unix:///var/run/docker.sock" \
    BACKUP_DIR="/data/backups" \
    DISABLE_BINARY_SELF_UPDATE="true" \
    UPDATE_REPOSITORY="autunn/dockerCopilot" \
    UPDATE_CHANNEL="latest" \
    TZ="Asia/Shanghai"

VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["./dockerCopilot", "-healthcheck"]
# 设置启动命令
CMD ["./start.sh"]
