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
- 自动备份、更新检查、代理和应用设置
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

后端需要 Go 1.23+，前端需要 Node.js 22。先启动后端，再启动前端：



```bash
# 后端
cd backend && go run dockercopilot.go -f etc/dockerCopilot.local.yaml

# 前端
cd frontend && npm start
```
前端开发地址：<http://localhost:4200/manager>。后端默认监听 `127.0.0.1:12712`。


### Docker daemon 代理（可选）

Linux 本地 Docker Engine 可安装 helper辅助程序。安装后，在 `docker/docker-compose.yml` 的 `volumes` 下增加：

```yaml
- /run/dockercopilot-helper.sock:/run/dockercopilot-helper.sock
```

然后执行：

```bash
docker compose -f docker/docker-compose.yml up -d
```

## 环境变量

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
| `HTTP_PROXY` | 空 | 否 | 应用 HTTP 客户端代理 |
| `HTTPS_PROXY` | 空 | 否 | 应用 HTTPS 客户端代理 |
| `NO_PROXY` | `localhost,127.0.0.1,::1` | 否 | 不使用代理的地址列表 |

`HTTP_PROXY`、`HTTPS_PROXY` 和 `NO_PROXY` 只影响 Docker Copilot 自身的 HTTP 请求，不影响 Docker daemon 拉取镜像时使用的代理。Docker daemon 代理请使用上面的 helper 功能配置。
