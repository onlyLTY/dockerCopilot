# Docker Copilot

[![License: AGPLv3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0.en.html)

Docker Copilot 是一个面向 Docker Engine 的 Web 管理平台，集中管理容器、镜像、Compose 项目、备份、端口和后台任务。

当前提供 `linux/amd64` 和 `linux/arm64` 镜像构建。



## 配置

### Compose 变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `secretKey` | 必填 | 登录密钥和 JWT 签名密钥，变量名区分大小写 |
| `DOCKER_COPILOT_IMAGE` | `dockercopilot:latest` | 要运行的镜像地址和标签 |
| `DOCKER_COPILOT_CONTAINER` | `dockercopilot` | 容器名称 |
| `DOCKER_BIND_ADDRESS` | `127.0.0.1` | 宿主机监听地址 |
| `DOCKER_PORT` | `12712` | 宿主机映射端口 |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | Docker socket 宿主机路径 |
| `DOCKER_DATA_DIR` | `./data` | `/data` 的宿主机目录 |
| `DOCKER_LOG_DIR` | `./logs` | `/app/logs` 的宿主机目录 |
| `DOCKER_COMPOSE_DIR` | `./compose` | `/compose` 的宿主机 Compose 根目录 |
| `TZ` | `Asia/Shanghai` | 容器时区 |



### 可选代理变量

代理也可以在管理台的“关于 → 设置”弹窗中填写。保存后需要重启 Docker Copilot 服务生效；Docker daemon 拉取镜像使用的代理仍需单独配置 daemon。

没有代理时保持 Compose 文件中的以下配置注释状态即可：

```yaml
# githubProxy: ${githubProxy:-}
# HTTP_PROXY: ${HTTP_PROXY:-}
# HTTPS_PROXY: ${HTTPS_PROXY:-}
# NO_PROXY: ${NO_PROXY:-localhost,127.0.0.1,::1}
```

需要代理时，取消对应注释并设置值：

- `githubProxy`：应用拼接 GitHub 版本和更新下载地址时使用的代理前缀。
- `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY`：标准 HTTP 客户端使用的代理环境变量。

### 挂载目录

Compose 文件位于仓库的 `docker/` 目录，默认挂载源使用 `../`，因此从仓库根目录看，数据、日志和 Compose 项目目录分别是 `./data`、`./logs` 和 `./compose`。

- `/data`：备份、图标、任务进度和应用设置。后端默认备份目录是 `/data/backups`，设置和任务状态位于 `/data/config`。
- `/app/logs`：服务日志。后端默认日志目录是 `./logs`，容器工作目录为 `/app`，因此对应 `/app/logs`。
- `/compose`：Compose 项目扫描、编辑和部署的根目录，对应 `backend/etc/dockerCopilot.yaml` 中的 `Compose.ScanPaths`。
- `/var/run/docker.sock`：Docker Engine 管理接口。应用需要通过它读取和管理宿主机 Docker 资源。

直接挂载 Docker socket 等价于授予容器较高的宿主机 Docker 管理权限。请限制管理端口来源，只在可信环境使用；生产环境可考虑使用 Docker socket 代理或其他访问控制方案。

## 功能概览

### 认证与界面

- 使用 `secretKey` 登录管理台并签发 JWT。
- 除登录接口外，管理 API 默认要求认证。
- 支持浅色/深色主题、紧凑显示模式和移动端菜单布局。

### 容器管理

- 查看容器名称、状态、镜像、端口和运行信息。
- 按全部、运行中、已停止、有更新和已忽略更新筛选。
- 启动、停止、重启和重命名容器。
- 按指定镜像 Tag 更新容器，支持批量更新和一键更新。
- 忽略或恢复指定容器的镜像更新检查。
- 查看更新检查和更新操作的异步进度。
- 支持手动检查镜像更新，也支持按计划自动检查。

### 镜像管理

- 查看镜像名称、Tag、大小、创建时间和使用状态。
- 按全部、使用中、未使用和无 Tag 筛选镜像。
- 删除普通镜像。
- 清理无 Tag 镜像或未使用镜像。
- 对有可用仓库摘要的镜像执行版本检查。

### 备份与恢复

- 创建容器配置 JSON 备份。
- 查看、筛选和删除备份文件。
- 异步恢复 JSON 容器配置备份。
- 将当前容器配置导出为 YAML/Compose 文件。
- 配置自动备份频率和备份保留数量。
- YAML/Compose 导出用于保存或迁移参考，不是可直接恢复的完整项目快照；当前恢复流程以 JSON 备份为准。

### Compose 项目管理

- 扫描并查看挂载目录中的 Compose 项目。
- 创建项目、读取和编辑项目文件。
- 校验 YAML/Compose 配置并反馈服务信息。
- 创建项目后部署，或对已有项目重新部署。
- 部署前查看预览并确认风险，部署过程显示异步任务进度。
- 支持部署时重新拉取镜像。
- 预览并清理未使用的 Compose 项目。
- 汇总 Compose 项目的容器和端口信息。



### 端口、任务与日志

- 汇总所有容器的端口映射，显示宿主机端口、容器端口和协议。
- 检测端口冲突，并支持按冲突状态筛选。
- 集中查看容器更新、备份恢复和 Compose 部署等后台任务。
- 查看任务阶段、进度、状态和详细信息。
- 查看最近服务日志，并按 DEBUG、INFO、WARN、ERROR 等级筛选。

### 图标与应用设置

- 上传、替换、查看和删除自定义镜像图标。
- 支持 PNG、JPEG、WebP 和 SVG 图标。
- 设置镜像更新检查频率、自动备份频率、日志级别和备份保留数量。
- 在“关于 → 设置”中配置 GitHub 地址前缀代理、HTTP/HTTPS 代理和 NO_PROXY。
- 代理设置保存到 `/data/config/appSettings.json`，重启 Docker Copilot 后生效；Docker daemon 的镜像拉取代理需要单独配置。
- 保存容器更新忽略名单。
- 查看本地/远程版本信息，并下载程序更新；更新文件会在服务重启时切换生效。

## 开发环境

### 前端

前端使用 Angular，CI 使用 Node.js 22。开发时需要后端服务监听 `127.0.0.1:12712`，前端代理会把 `/api` 请求转发到该地址。

```bash
cd frontend && npm start
npm ci

```

开发服务器启动后访问 <http://localhost:4200/manager>。生产构建：

```bash
cd frontend && npm run build
```

产物位于仓库根目录 `build/frontend/browser/`。后端运行时从工作目录下的 `web/` 目录读取前端静态文件。

### 后端

后端模块要求 Go 1.23，`backend/go.mod` 声明的 toolchain 为 Go 1.24.1。运行测试：

```bash
cd backend
$env:secretKey="test123456"
go run dockercopilot.go -f etc/dockerCopilot.yaml
```

后端启动前需要准备配置文件 `backend/etc/dockerCopilot.yaml`、前端 `web/` 目录和可写的数据/日志目录。服务默认监听 `12712`，配置中的 `Auth.AccessSecret` 使用环境变量 `secretKey` 展开。

## 本地 Docker 构建

发布镜像模式不需要本地源码构建。需要验证本地 Dockerfile 时，在仓库根目录执行：

```bash
sh docker/build.sh docker
```

脚本默认按 Docker 容器目标平台构建 Linux 二进制，目标系统固定为 `linux`，不会受到 Windows `OS=Windows_NT` 环境变量影响。默认架构取当前 Go 环境的 `GOARCH`；如果需要指定架构，使用 `TARGET_ARCH`：

```bash
TARGET_ARCH=amd64 sh docker/build.sh docker
```

然后继续构建镜像并启动：

```bash
docker build -f docker/Dockerfile \
  --build-arg TARGETPLATFORM=linux/amd64 \
  -t dockercopilot:local .
DOCKER_COPILOT_IMAGE=dockercopilot:local \
secretKey='替换为强密码' \
  docker compose -f docker/docker-compose.yml up -d
```

ARM64 构建需要让 Go 架构、Dockerfile 参数和目标镜像一致：

```bash
TARGET_ARCH=arm64 sh docker/build.sh docker
docker build -f docker/Dockerfile \
  --build-arg TARGETPLATFORM=linux/arm64 \
  -t dockercopilot:local .
```

## 当前边界

以下能力不属于当前实现范围：

- 多用户、角色权限和 RBAC。
- 完整 Docker Compose 语法兼容。
- YAML 备份直接恢复为完整 Compose 项目。
- 无缝升级或升级失败自动回滚。
- 远程备份、备份加密和镜像仓库凭据管理。
- 跨多个 Docker daemon 的统一管理。

## 许可证

本项目使用 [AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html) 许可证。
