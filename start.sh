#!/bin/sh
set -eu

APP_DIR=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$APP_DIR"
# 判断当前目录下是否存在名为 dockerCopilot-new 的二进制文件
if [ -f "./dockerCopilot-new" ]; then
    # 新文件必须先具备安全的执行权限，避免替换后因 chmod 失败而无法启动。
    chmod 0700 ./dockerCopilot-new
    # 先保留当前版本；安装动作失败时立即恢复，避免留下不可启动的目录。
    if [ -f "./dockerCopilot" ]; then
        mv -f ./dockerCopilot ./dockerCopilot-old
        if ! mv -f ./dockerCopilot-new ./dockerCopilot; then
            mv -f ./dockerCopilot-old ./dockerCopilot
            exit 1
        fi
    else
        mv -f ./dockerCopilot-new ./dockerCopilot
    fi
fi

# 运行 dockerCopilot
exec ./dockerCopilot
