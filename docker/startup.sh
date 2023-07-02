#!/bin/sh
printenv | sed 's/^\(.*\)$/export \1/g' > /root/project_env.sh
python /app/manage.py check_update_command >> /var/log/cron.log 2>&1
cron && tail -f /var/log/cron.log

