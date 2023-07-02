#!/bin/sh
printenv | sed 's/^\(.*\)$/export \1/g' > /root/project_env.sh
cron && tail -f /var/log/cron.log

