FROM alpine
LABEL authors="onlyLTY"
WORKDIR /app

COPY . .
RUN chmod +x start.sh

ENV secretKey="" \
    hubURL="https://docker.nju.edu.cn"

CMD ["./start.sh"]