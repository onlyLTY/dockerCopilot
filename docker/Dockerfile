FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

COPY . .
RUN chmod +x start.sh

ENV account="" \
    secretKey="" \
    hubURL="https://hub.docker.com"

# 指定容器启动时运行的命令
ENTRYPOINT ["./start.sh"]