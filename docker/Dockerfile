# FROM python:slim or python:alpine 以精简镜像，但需要额外下载依赖
FROM python:3.11.2

# 使用清华源以提高下载速度
RUN pip install --no-cache-dir -r https://raw.githubusercontent.com/onlyLTY/oneKeyUpdate/web/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple/

COPY startup.sh /startup.sh
COPY mycron /etc/cron.d/mycron

ENV LANG="C.UTF-8" \
    TZ="Asia/Shanghai" \
    REPO_URL="https://ghproxy.com/https://github.com/onlyLTY/oneKeyUpdate.git" \
    WORKDIR="/app" \
    AUTO_UPDATE="true" \
    account="" \
    SECRET_KEY=""

WORKDIR ${WORKDIR}

RUN apt-get update \
    && apt-get install -y cron \
    && git clone -b web ${REPO_URL} ${WORKDIR} --depth=1 --recurse-submodule \
    && chmod +x ${WORKDIR}/entrypoint.sh \
    && chmod +x /startup.sh \
    && chmod 0644 /etc/cron.d/mycron

EXPOSE 12712

ENTRYPOINT ["/app/entrypoint.sh"]