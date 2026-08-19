# Docker Copilot

Docker Copilot 是一个面向 Docker Engine 的 Web 管理平台，用于在浏览器中管理容器、镜像、Compose 项目、备份、端口和后台任务。

## 功能

- 单密钥登录与 JWT 会话认证
- 容器查看、启动、停止、重启、重命名和批量更新
- 镜像查看、版本检查、删除和清理
- 容器配置备份、恢复，以及导出为 Compose YAML 参考
- Compose 项目扫描、编辑、预览、部署和风险检查
- 端口映射汇总与冲突检测
- 后台任务进度、服务日志和自定义图标
- 自动备份、更新检查、GitHub 更新代理和应用设置
- 支持 `linux/amd64` 与 `linux/arm64`

## 使用方法

### Docker 部署

需要 Docker Engine 和 Docker Compose。将以下内容保存为环境变量后启动：

```bash
export secretKey='替换为至少 8 位的强密码'
docker compose -f docker/docker-compose.yml up -d
```

访问：<http://127.0.0.1:12712/manager>

Docker Copilot 需要访问 Docker socket，请仅在可信环境中使用，并限制管理端口的访问来源。

### 本地开发

后端需要 Go 1.23+，前端需要 Node.js 22。先从配置模板生成本地配置文件，再启动后端和前端：

```bash
# 生成本地配置文件，并设置本地开发所需的登录密钥
cp backend/etc/dockerCopilot.yaml backend/etc/dockerCopilot.local.yaml

# 启动后端
cd backend && go run dockercopilot.go -f etc/dockerCopilot.local.yaml

# 另开一个终端启动前端
cd frontend && npm start
```

如需修改监听地址、数据目录或 Compose 扫描目录，请编辑复制出来的 `backend/etc/dockerCopilot.local.yaml`。前端开发地址：<http://localhost:4200/manager>。后端默认监听 `127.0.0.1:12712`。

如果只需要检查前端真实页面、布局和交互，无需启动 Go 后端，可以使用 `subClash` 同样的预览构建模式：

```bash
cd frontend && npm run start:preview
```

然后访问 <http://127.0.0.1:4209/manager>。预览模式继续使用容器、镜像、Compose、备份、端口、任务和关于页面的现有 HTML 与样式，但通过本地 fixture 响应接口，不连接 Docker Engine；适合快速测试页面状态、弹窗、筛选和按钮交互。

### 缓存与刷新策略

- 容器页面在页面可见期间每 20 秒自动刷新一次；容器列表缓存有效期也统一为 20 秒。切换到其他页面后，容器页面的定时刷新会停止。
- 镜像、Compose 项目、端口和备份列表使用当前会话缓存，不会仅因经过一段时间就重复加载。
- 相关操作完成后会主动刷新受影响的数据：镜像清理刷新镜像；Compose 重新部署刷新容器、镜像、项目和端口；备份恢复刷新容器、镜像和端口。
- 手动刷新仍可随时重新读取当前页面的数据。页面切回可见时，容器页面会立即检查一次最新状态。

### 环境变量

以下变量可在启动前通过 `export` 设置，也可以写入本地 Compose 文件。默认值以 `docker/docker-compose.yml` 为准。

| 变量 | 默认值 | 必填 | 说明 |
| --- | --- | --- | --- |
| `secretKey` | 无 | 是 | 登录密钥和 JWT 签名密钥，至少 8 位且不能为纯数字 |
| `DOCKER_COPILOT_IMAGE` | `dockercopilot:latest` | 否 | 镜像地址与标签 |
| `DOCKER_COPILOT_CONTAINER` | `dockercopilot` | 否 | 容器名称 |
| `DOCKER_BIND_ADDRESS` | `127.0.0.1` | 否 | 宿主机监听地址；局域网访问可设为 `0.0.0.0` |
| `DOCKER_PORT` | `12712` | 否 | 宿主机映射端口 |
| `DOCKER_SOCKET` | `/var/run/docker.sock` | 否 | Docker socket 宿主机路径 |
| `DOCKER_DATA_DIR` | `../data` | 否 | 映射到容器 `/data`，保存备份、设置、图标和任务进度 |
| `DOCKER_LOG_DIR` | `../logs` | 否 | 映射到容器 `/app/logs`，保存服务日志 |
| `DOCKER_COMPOSE_DIR` | `../compose` | 否 | 映射到容器 `/compose`，作为 Compose 项目目录 |
| `TZ` | `Asia/Shanghai` | 否 | 容器时区 |
| `githubProxy` | 空 | 否 | GitHub 版本和更新下载地址的前缀代理 |

`githubProxy` 只用于 Copilot 的 GitHub 版本检查和程序更新地址。Docker 镜像拉取使用 Docker Engine 自身的 daemon 配置。
