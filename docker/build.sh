#!/bin/sh
# 本地完整构建脚本：前端 -> 后端二进制 -> Docker 镜像上下文暂存
# 前端不再嵌入二进制，运行时由后端从磁盘 ./web 目录读取。
# 用法:
#   docker/build.sh            仅构建 Linux 后端二进制（默认当前架构）
#   docker/build.sh docker     额外准备 Docker 镜像上下文（build/docker）
# 脚本位于 docker/ 下，ROOT 上溯一级指向仓库根。
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 目标平台：Docker 镜像使用 Linux；TARGET_ARCH 优先，ARCH 保留兼容性
TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-${ARCH:-$(cd backend && go env GOARCH)}}"

echo "==> 目标平台: $TARGET_OS/$TARGET_ARCH"

echo "==> [1/3] 构建前端 (Angular)"
cd "$ROOT/frontend"
npm ci
npm run build
# angular.json 的 outputPath = ../build/frontend，产物在 build/frontend/browser

echo "==> [2/3] 构建后端二进制 (build/backend/$TARGET_OS/$TARGET_ARCH)"
VERSION="$(cat "$ROOT/version")"
OUT="$ROOT/build/backend/$TARGET_OS/$TARGET_ARCH"
mkdir -p "$OUT"
cd "$ROOT/backend"
CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -a --trimpath \
  -ldflags="-w -s -X 'github.com/onlyLTY/dockerCopilot/internal/config.Version=${VERSION}' -X 'github.com/onlyLTY/dockerCopilot/internal/config.BuildDate=$(date)'" \
  -o "$OUT/dockerCopilot" .

if [ "$1" = "docker" ]; then
  echo "==> [3/3] 准备 Docker 镜像上下文 (build/docker)"
  DOCKER_CTX="$ROOT/build/docker"
  rm -rf "$DOCKER_CTX"
  mkdir -p "$DOCKER_CTX/dist/$TARGET_OS/$TARGET_ARCH" "$DOCKER_CTX/etc" "$DOCKER_CTX/web"
  cp "$OUT/dockerCopilot" "$DOCKER_CTX/dist/$TARGET_OS/$TARGET_ARCH/dockerCopilot"
  cp -r "$ROOT/backend/etc/." "$DOCKER_CTX/etc/"
  cp -r "$ROOT/build/frontend/browser/." "$DOCKER_CTX/web/"
  echo "    完成。可用: docker build -f docker/Dockerfile --build-arg TARGETPLATFORM=$TARGET_OS/$TARGET_ARCH -t dockercopilot:local ."
else
  echo "==> 跳过 Docker 上下文（传入参数 docker 可生成）"
fi

echo "==> 构建完成: $OUT/dockerCopilot"
