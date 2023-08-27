#!/bin/sh

# 判断当前目录下是否存在名为 onekeyupdate-new 的二进制文件
if [ -f "./onekeyupdate-new" ]; then
    # 如果存在，则用它覆盖 onekeyupdate
    mv ./onekeyupdate-new ./onekeyupdate
    # 赋予 onekeyupdate 执行权限
    chmod +x ./onekeyupdate
fi

# 运行 onekeyupdate
./onekeyupdate