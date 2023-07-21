编译命令：
x86 'docker build --no-cache -t 0nlylty/one-key-update:latest . --push''
arm64 'docker buildx build --platform linux/arm64 --no-cache -t 0nlylty/one-key-update:arm64 . --push'