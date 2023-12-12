#!/bin/sh
cd "${WORKDIR}" || exit
# 判断当前目录下是否存在名为 dockerCopilot-new 的二进制文件
if [ -f "./dockerCopilot-new" ]; then
    # 如果存在，则用它覆盖 dockerCopilot
    mv ./dockerCopilot-new ./dockerCopilot
    # 赋予 dockerCopilot 执行权限
    chmod +x ./dockerCopilot
fi

# 运行 dockerCopilot
./dockerCopilot
