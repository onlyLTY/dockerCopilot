# 使用 Alpine 作为基础镜像
FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

ARG TARGETPLATFORM
# 复制适当的平台二进制文件
COPY dist/$TARGETPLATFORM/onekeyupdate /app/onekeyupdate

# 复制其他需要的文件或目录，例如 start.sh
COPY start.sh /app/start.sh
COPY etc /app/etc
RUN chmod +x /app/start.sh

# 设置环境变量
ENV secretKey="" \
    hubURL="https://docker.nju.edu.cn"

# 设置启动命令
CMD ["./start.sh"]