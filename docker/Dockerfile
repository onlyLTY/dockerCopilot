FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

COPY . .

ENV secretKey="" \
    hubURL="https://docker.nju.edu.cn"

CMD ["./onekeyupdate"]