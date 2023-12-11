#!/bin/sh
cd "${WORKDIR}" || exit
# 判断当前目录下是否存在名为 dokcerCopilot-new 的二进制文件
if [ -f "./dokcerCopilot-new" ]; then
    # 如果存在，则用它覆盖 dokcerCopilot
    mv ./dokcerCopilot-new ./dokcerCopilot
    # 赋予 dokcerCopilot 执行权限
    chmod +x ./dokcerCopilot
fi

# 运行 dokcerCopilot
./dokcerCopilot
