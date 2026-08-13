# Docker Copilot

[![License: AGPLv3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0.en.html)

## 1. 介绍

Docker Copilot 是一个面向 Docker Engine 的 Web 管理平台，用于在浏览器中集中管理容器、镜像、Compose 项目、备份、端口映射与后台任务。

**主要特点：**

- 单密钥登录（`secretKey`），管理 API 默认需要认证
- 容器启停、重命名、镜像更新检查与批量更新
- 镜像列表、清理与版本检查
- 容器配置备份 / 恢复，以及导出为 Compose YAML 参考
- Compose 项目扫描、编辑、预览部署与风险门禁
- 端口汇总、任务进度、服务日志与自定义图标
- 提供 `linux/amd64` 与 `linux/arm64` 镜像构建

> **安全提示：** 挂载 Docker socket 等价于授予容器较高的宿主机 Docker 管理权限。请限制管理端口来源，只在可信环境使用。

---

## 2. 部署方式

### 2.1 本地开发运行

适合改代码、调试 API 与前端。需要本机已安装 Go、Node.js，以及可访问的 Docker Engine。

#### 后端

后端基于 go-zero，遵循 **Spec-First**（改 `backend/dockercopilot.api` → `goctl api go --style gozero` → 填 `internal/logic`）。详见 `backend/README.md`。

后端模块要求 Go 1.23+（`backend/go.mod` 声明 toolchain 为 Go 1.24.1）。

- **主配置**：`backend/etc/dockerCopilot.yaml`（容器/通用；`AccessSecret: ${secretKey}` 需环境变量；`DataDir` 可省略，容器挂 `/data` 即可）
- **本地固定配置（推荐）**：复制一份 `backend/etc/dockerCopilot.local.yaml`（已 gitignore），把密钥、`DataDir`、Compose 路径写死，**不必每次 export**

```bash
# 首次（若还没有 local 文件）
cp backend/etc/dockerCopilot.yaml backend/etc/dockerCopilot.local.yaml
# 编辑 local：AccessSecret 明文；DataDir: ../docker/test/data；Compose.ScanPaths 改成本机目录

cd backend && go run dockercopilot.go -f etc/dockerCopilot.local.yaml
```

仍可用主配置 + 环境变量：

```bash
cd backend
export secretKey='test123456'   # PowerShell: $env:secretKey="test123456"
# 可选：export DATA_DIR=... 覆盖 yaml DataDir
go run dockercopilot.go -f etc/dockerCopilot.yaml
```

- 默认监听 `12712`
- **数据根目录**优先级：`DATA_DIR` 环境变量 > yaml `DataDir` > 默认 `/data`。其下：`config/appSettings.json`（设置/加速源）、`backups/`（**容器备份页**）、`icon/`。仍可用 `BACKUP_DIR`、`APP_SETTINGS_PATH`、`TASK_PROGRESS_PATH` 单独覆盖
- **Compose 路径**在 yaml：`Compose.ScanPaths` 扫描项目；
- 健康检查：`GET /healthz`（不鉴权；会 Ping Docker，不可达时返回 503）


#### 前端

前端使用 Angular，CI 使用 Node.js 22。开发时后端需监听 `127.0.0.1:12712`，前端通过代理转发 `/api`。

```bash
# 安装依赖
cd frontend && npm start
```

- 开发地址：<http://localhost:4200/manager>
- 生产构建：`cd frontend && npm run build`
- 产物在 `build/frontend/browser/`；后端运行时从工作目录下的 `web/` 读取静态资源

---

### 2.2 Docker 部署（推荐）

仓库内只维护一份 Compose：[`docker/docker-compose.yml`](docker/docker-compose.yml)。

#### 快速启动

```bash
# 在仓库根目录；secretKey 必填
export secretKey='替换为强密码'
docker compose -f docker/docker-compose.yml up -d
```

自用想把密钥/挂载写进文件时，复制一份本地文件即可（已 gitignore，不会提交）：

```bash
cp docker/docker-compose.yml docker/docker-compose.local.yml
# 编辑 local：把 secretKey: ${secretKey:?...} 改成明文，挂载可改成 ./test/...
docker compose -f docker/docker-compose.local.yml up -d
```

浏览器访问：<http://127.0.0.1:12712/manager>（默认仅绑定本机 `127.0.0.1`）。

查看状态 / 日志 / 停止（`-f` 与 up 时用的文件一致即可）：

```bash
docker compose -f docker/docker-compose.yml ps
docker compose -f docker/docker-compose.yml logs -f
docker compose -f docker/docker-compose.yml down
```

**升级提示：** 只换镜像/容器即可，**不要删除** `DOCKER_DATA_DIR` 挂载的数据目录（备份、设置、任务进度在 `/data`）。

#### 环境变量一览

默认值写在 `docker/docker-compose.yml` 的 `${VAR:-默认值}` 中；启动前 `export` 可临时覆盖，或直接改 compose 文件。

`DOCKER_DATA_DIR` / `DOCKER_LOG_DIR` / `DOCKER_COMPOSE_DIR` 只影响**宿主机哪条目录挂进容器**，应用仍写容器内固定路径（`/data`、`/app/logs`、`/compose`）。

| 变量 | 默认值 | 必填 | 说明 |
| --- | --- | --- | --- |
| `secretKey` | 无 | **是** | 登录密钥与 JWT 签名密钥；至少 8 位且不能为纯数字（启动强制校验） |
| `DOCKER_COPILOT_IMAGE` | `dockercopilot:latest` | 否 | 镜像地址与标签 |
| `DOCKER_COPILOT_CONTAINER` | `dockercopilot` | 否 | 容器名称 |
| `DOCKER_BIND_ADDRESS` | `127.0.0.1` | 否 | 宿主机监听地址；局域网访问改为 `0.0.0.0` 并做好隔离 |
| `DOCKER_PORT` | `12712` | 否 | 宿主机映射端口 |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | 否 | Docker socket 宿主机路径 |
| `DOCKER_DATA_DIR` | `../data`（相对 `docker/`） | 否 | 映射到容器 `/data`；从仓库根看为 `./data` |
| `DOCKER_LOG_DIR` | `../logs`（相对 `docker/`） | 否 | 映射到容器 `/app/logs`；从仓库根看为 `./logs` |
| `DOCKER_COMPOSE_DIR` | `../compose`（相对 `docker/`） | 否 | 映射到容器 `/compose`；从仓库根看为 `./compose` |
| `TZ` | `Asia/Shanghai` | 否 | 容器时区 |

**可选代理变量**（默认在 compose 中注释；需要时取消注释并赋值）：

| 变量 | 说明 |
| --- | --- |
| `githubProxy` | 应用拼接 GitHub 版本/更新下载地址时的前缀代理 |
| `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` | 标准 HTTP 客户端代理（**不影响** Docker daemon 拉镜像代理） |

代理也可在管理台「关于 → 设置」中填写，保存后需重启 Docker Copilot 服务生效。

#### Docker daemon 代理 helper（Linux 高级功能）

管理台中的 GitHub 地址前缀代理只影响 Docker Copilot 自身。Docker 镜像拉取使用 Docker daemon 的代理。若要从管理台读取并修改 daemon 代理，可在 **Linux 本机 Docker Engine** 上安装 `helper/README.md` 中的受控 helper，然后显式启用覆盖文件：

```bash
docker compose \
  -f docker/docker-compose.yml \
  -f docker/docker-compose.daemon-helper.yml \
  up -d
```

默认部署不会挂载 helper socket，也不会提供修改宿主机 daemon 配置或重启 Docker 的按钮。启用 helper 后，页面会显示 daemon 当前生效代理和 `daemon.json` 待生效配置；“覆写 daemon 代理”和“重启 Docker daemon”均需要危险确认。重启会短暂影响宿主机上的所有 Docker 容器操作。该功能不支持 Docker Desktop、Windows、远程 daemon 或 rootless Docker。

**示例：自定义端口与挂载目录**

```bash
export secretKey='替换为强密码'
export DOCKER_PORT=18080
export DOCKER_DATA_DIR=/opt/dockercopilot/data
export DOCKER_LOG_DIR=/opt/dockercopilot/logs
export DOCKER_COMPOSE_DIR=/opt/dockercopilot/compose
docker compose -f docker/docker-compose.yml up -d
```

#### 挂载目录说明

Compose 中的相对路径**相对 `docker/docker-compose.yml` 所在目录**解析，不是相对你执行命令时的当前目录。

| 环境变量 | 默认宿主机路径（相对 `docker/`） | 容器内路径 | 用途 |
| --- | --- | --- | --- |
| `DOCKER_DATA_DIR` | `../data` → 仓库根 `./data` | `/data` | 备份、图标、任务进度、应用设置；备份默认 `/data/backups`，配置与 `taskProgress.json` 在 `/data/config` |
| `DOCKER_LOG_DIR` | `../logs` → 仓库根 `./logs` | `/app/logs` | 服务日志（程序默认写 `./logs`，容器工作目录为 `/app`） |
| `DOCKER_COMPOSE_DIR` | `../compose` → 仓库根 `./compose` | `/compose` | Compose 项目扫描/编辑/部署根目录，对应配置 `Compose.ScanPaths` |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | `/var/run/docker.sock` | 访问 Docker Engine |

对应 compose 片段：

```yaml
volumes:
  - ${DOCKER_SOCKET:-/var/run/docker.sock}:/var/run/docker.sock
  - ${DOCKER_DATA_DIR:-../data}:/data
  - ${DOCKER_LOG_DIR:-../logs}:/app/logs
  - ${DOCKER_COMPOSE_DIR:-../compose}:/compose
```

**挂载可以自定义：** 设 `DOCKER_DATA_DIR` / `DOCKER_LOG_DIR` / `DOCKER_COMPOSE_DIR` 即可改宿主机目录，容器内路径不用动。默认数据目录已 gitignore；生产可指到如 `/var/lib/dockercopilot/...`。

#### 本地构建镜像再部署

发布场景一般直接拉镜像；若要验证本仓库 Dockerfile：

```bash
# 1）构建 Linux 二进制（目标 OS 固定 linux，可用 TARGET_ARCH=amd64|arm64）
sh docker/build.sh docker

# 2）构建镜像
docker build -f docker/Dockerfile \
  --build-arg TARGETPLATFORM=linux/amd64 \
  -t dockercopilot:local .

# 3）用本地镜像启动
DOCKER_COPILOT_IMAGE=dockercopilot:local \
secretKey='替换为强密码' \
  docker compose -f docker/docker-compose.yml up -d
```

ARM64 时保持 Go 架构、Dockerfile 参数与镜像一致：

```bash
TARGET_ARCH=arm64 sh docker/build.sh docker
docker build -f docker/Dockerfile \
  --build-arg TARGETPLATFORM=linux/arm64 \
  -t dockercopilot:local .
```

#### Compose 变量插值

部署/解析时 `${VAR}` **只**从该项目目录的 `.env` 读取，**不会**读取 Docker Copilot 进程环境（避免 `secretKey` 等被写进业务容器）。业务所需变量请写在项目 `.env` 或 compose 字面量中。

#### Compose 风险控制（`AllowHighRisk`）

部署用户 Compose 项目前会检查风险项。极高危配置（如 `privileged`、挂载 `docker.sock`、敏感宿主机路径）**默认拒绝**；仅当服务端 `backend/etc/dockerCopilot.yaml`（镜像内 `etc/dockerCopilot.yaml`）中：

```yaml
Compose:
  AllowHighRisk: false   # 仅在可信环境、确需特权时改为 true，并重启服务
```

说明：前端确认无法绕过；开启后仍走预览与确认。备份恢复与 Compose 部署使用同一套高危规则：默认拒绝 privileged、docker.sock、敏感路径，以及 host 网络/PID、devices、cap_add、security_opt；仅 `AllowHighRisk: true` 可放行。

---

## 3. 功能概述

### 认证与界面

- 使用 `secretKey` 登录并签发 JWT；启动要求至少 8 位且不能为纯数字；除登录外管理 API 默认需认证
- 登录按客户端 IP 限制失败次数（约 5 次/分钟后短暂封禁；状态落盘至 `/data/config/loginAttempts.json`，重启仍生效）；会话过期（401）时提示并保留 `returnUrl` 回到原页面
- Token 有效期由 `Auth.AccessExpire` 配置（默认 7 天）；共享设备请自行缩短并保护 `secretKey`
- 浅色/深色主题、紧凑模式、移动端菜单
- 全局安全响应头（CSP / X-Frame-Options 等）

### 容器管理

- 列表：名称、状态、镜像、端口等；筛选全部 / 运行中 / 已停止 / 有更新 / 已忽略
- 启动、停止、重启、重命名
- 按 Tag 更新、批量更新、一键更新；忽略或恢复更新检查
- 手动或按计划检查镜像更新；查看异步进度

### 镜像管理

- 列表与筛选（全部 / 使用中 / 未使用 / 无 Tag）
- 删除镜像；清理无 Tag 或未使用镜像
- 对有仓库摘要的镜像做版本检查

### 备份与恢复

- 创建容器配置 JSON 备份；列表、筛选、删除
- 异步恢复 JSON 备份；导出 YAML/Compose 作迁移参考（恢复以 JSON 为准）
- 自动备份频率与保留数量可配
- 恢复门禁与 Compose 部署对齐：默认拒绝 privileged / docker.sock / 敏感挂载，以及 host 网络·PID、devices、cap_add、security_opt（`AllowHighRisk` 可放行）

### Compose 项目管理

- 扫描挂载目录中的项目；创建、读写与校验
- 部署预览、风险提示、异步进度；可重新拉镜像
- 清理未使用项目；汇总容器与端口
- 极高危配置默认拦截（见上文 `AllowHighRisk`）

### 端口、任务与日志

- 端口映射汇总与冲突检测
- 后台任务（更新、恢复、Compose 部署）集中查看；`/api/progress/list` 批量轮询
- 任务进度带 `updatedAt`，持久化在 `/data/config/taskProgress.json`（进行中合并写盘，完成立即落盘；已完成按时间/条数淘汰）
- 服务日志按 DEBUG / INFO / WARN / ERROR 筛选
- 文件日志**按天轮转**（go-zero `Rotation=daily`），默认保留 7 天并压缩历史；目录 `/app/logs` 或本地 `./logs`

### 图标与应用设置

- 自定义镜像图标（仅 PNG）
- 更新检查频率、自动备份、日志级别、代理、忽略名单等
- 本地/远程版本信息与程序更新（官方域名白名单 + 更新包 SHA256 校验；重启后切换生效）

### 当前边界

以下能力不在当前范围：多用户/RBAC、完整 Compose 语法兼容、YAML 备份直接恢复为完整项目、无缝升级与自动回滚、远程备份/加密、镜像仓库凭据、多 Docker daemon 统一管理。

---

## 许可证

本项目使用 [AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html) 许可证。
