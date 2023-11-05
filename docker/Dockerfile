# 使用 Alpine 作为基础镜像
FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

ARG TARGETPLATFORM
# 复制适当的平台二进制文件
COPY dist/$TARGETPLATFORM/onekeyupdate /app/onekeyupdate

# 复制其他需要的文件或目录，例如 start.sh
COPY . ${WORKDIR}
RUN chmod +x start.sh

ENV account="" \
    secretKey="" \
    hubURL="https://docker.nju.edu.cn" \
    githubProxy="https://gh-proxy.com/" \

VOLUME ["/data"]
# 指定容器启动时运行的命令
ENTRYPOINT ["./start.sh"]