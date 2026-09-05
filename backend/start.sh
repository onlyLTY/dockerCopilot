#!/bin/sh
cd "${WORKDIR}" || exit
# 判断当前目录下是否存在名为 dockerCopilot-new 的二进制文件
if [ -f "./dockerCopilot-new" ]; then
    # 如果存在，则用它覆盖 dockerCopilot
    mv ./dockerCopilot-new ./dockerCopilot
    # 赋予 dockerCopilot 执行权限
    chmod +x ./dockerCopilot
fi

# 判断是否存在待更新的前端目录 web-new，存在则整体替换 web
# 与二进制在同一次重启一起切换，避免前后端版本不一致
if [ -d "./web-new" ]; then
    rm -rf ./web
    mv ./web-new ./web
fi

# 同步切换 version 文件（若更新包携带），保证自更新后版本号上报正确
if [ -f "./version-new" ]; then
    mv ./version-new ./version
fi

# 运行 dockerCopilot
./dockerCopilot
