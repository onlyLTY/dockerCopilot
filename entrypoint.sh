#!/bin/sh
cd ${WORKDIR}

# 自动更新
if [ "${AUTO_UPDATE}" = "true" ]; then
    if [ ! -s /tmp/requirements.txt.sha256sum ]; then
        sha256sum /app/requirements.txt > /tmp/requirements.txt.sha256sum
    fi
    echo "更新源码"
    branch=web
    git clean -dffx
    git fetch --depth 1 origin ${branch}
    git reset --hard origin/${branch}
    if [ $? -eq 0 ]; then
        echo "源码更新成功"
        chmod +x ${WORKDIR}/entrypoint.sh
        echo "Python更新依赖包"
        hash_old=$(cat /tmp/requirements.txt.sha256sum)
        hash_new=$(sha256sum requirements.txt)
        if [ "${hash_old}" != "${hash_new}" ]; then
            echo "检测到requirements.txt有变化，重新安装依赖..."
            pip install -r /app/requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple/
            if [ $? -ne 0 ]; then
                echo "无法安装依赖，请更新镜像..."
            else
                echo "依赖安装成功..."
                sha256sum /app/requirements.txt > /tmp/requirements.txt.sha256sum
            fi
        fi
    else
        echo "更新失败"
    fi
else
    echo "程序自动升级已关闭，如需自动升级请在创建容器时设置环境变量：AUTO_UPDATE=true"
fi
# 启动主程序
# 迁移数据库
python /app/manage.py migrate
# 启动cron
service cron start
service cron status
# 添加定时任务
# python /app/manage.py crontab add
# python /app/manage.py crontab show
# python /app/update/update.py &
# 复制cron
cp /app/docker/mycron /etc/cron.d/
chmod 0644 /etc/cron.d/mycron
cron
tail -f /var/log/cron.log &
/startup.sh
python /app/manage.py runserver \[::\]:12712 >> /app/run.log 2>&1 &
python /app/manage.py check_update_command >> /var/log/cron.log 2>&1 &
