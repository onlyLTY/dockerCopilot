# dockerCopilot Backend Service

dockerCopilot 后端服务基于 [go-zero](https://github.com/zeromicro/go-zero) 框架开发，遵循 Spec-First 规范架构设计。

## 目录架构

```
backend/
├── dockercopilot.api        # API 规范描述文件 (Spec-First)
├── dockercopilot.go         # 服务入口文件
├── etc/
│   └── dockerCopilot.yaml   # 配置文件
├── internal/
│   ├── config/              # 配置结构体与版本解析
│   ├── errorx/              # 统一 CodeError
│   ├── handler/             # HTTP 路由与 Handler（含 goctl 生成与手写补丁）
│   ├── logic/               # 业务逻辑
│   ├── middleware/          # 登录限流、安全响应头等
│   ├── svc/                 # ServiceContext、任务进度、定时任务
│   ├── types/               # 请求/响应结构体（goctl 生成）
│   └── utiles/              # Docker/Compose/备份等工具与子包
└── README.md
```

## 核心开发规范

1. **Spec-First**：API 变更优先改 `dockercopilot.api`；types 可用 goctl 生成。`routes.go` 以手写维护为主，**不要**让 goctl 空 stub 直接挂路由。
2. **Handler 约定（goctl 风格）**：`Parse` → `NewXxxLogic` → `writeLogicResp` / `handler.WriteLogicResp`。Compose 复杂操作可聚合在 `ActionsLogic` / `FilesLogic`，但 **Handler 符号名** 与 api 中 `@handler` 对齐（如 `ComposeDeployHandler`）。
3. **错误处理**：业务错误统一用 `internal/errorx.CodeError`；全局 `SetErrorHandler` 只认该类型。Logic 失败时填好 `resp.Code/Msg` 并返回 err，由 handler 写出 HTTP 200 + body。
4. **Docker 客户端**：调用 Engine 前使用 `svcCtx.RequireDocker()` / `utiles` 内 `requireDocker`；不可用时业务码 **503**（`errorx.ErrDockerUnavailable`），禁止对 nil client 直接解引用。
5. **手写路由**：中间件与 settings/logs 等在 `routes_manual.go`。重新 goctl 后务必保留 `RegisterManualHandlers` 调用，并核对 Compose/镜像 Handler 未退回空 stub。
6. **日志**：`SetupLog` 使用 go-zero file 模式，**按天轮转**（`Rotation=daily`），`KeepDays=7` 并压缩历史。
7. **编译与验证**：`go test ./...` 与 `go build ./...`。

## 启动与运行

本地推荐固定配置（`etc/dockerCopilot.local.yaml` 已 gitignore，密钥写在文件里，无需 export）：

```bash
# 首次可从主配置复制后改 AccessSecret / ScanPaths
# cp etc/dockerCopilot.yaml etc/dockerCopilot.local.yaml
go run dockercopilot.go -f etc/dockerCopilot.local.yaml
```

或主配置 + 环境变量：

```bash
export secretKey=test123456
go run dockercopilot.go -f etc/dockerCopilot.yaml
go build ./...
go test ./...
```

- `etc/dockerCopilot.yaml`：通用/容器用，`Auth.AccessSecret: ${secretKey}`；`Timeout` 默认 10 分钟；`AccessExpire` 默认 7 天
- `etc/dockerCopilot.local.yaml`：本机开发，AccessSecret 可写明文；`Compose.AllowHighRisk` 控制极高危部署门禁
- 进程退出：`proc` WrapUp 刷任务进度，Shutdown/`defer` 停 cron 并关闭 Docker 客户端

## 核心接口说明

### 认证

- `POST /api/auth` — 登录（含 IP 失败限流）

### 容器

- `GET /api/containers` — 容器列表
- `POST /api/containers/check-update` — 检查镜像更新（异步任务）
- `POST /api/container/:id/start|stop|restart|rename|update`
- `POST|DELETE /api/container/:id/update-ignore` — 忽略/恢复更新检测
- 备份：`/api/container/backup`、`listBackups`、`backups/restore` 等

### Compose

- `GET|POST /api/compose/projects`
- `POST /api/compose/projects/:id/deploy` 与 `deploy/preview`
- 文件读写、cleanup、`POST /api/compose/validate`
- 实现在 `handler/compose` 的 actions/files/list 手写 handler；goctl 空 stub 已移除，重新生成后勿把 stub 再挂回 `routes.go`

### 镜像 / 端口 / 图标 / 进度 / 版本

- `GET /api/images`、`POST /api/images/prune`、`DELETE /api/image/:id`（JWT + `/api` 前缀）
- `GET /api/ports`
- `/api/icons`
- `GET /api/progress/list`、`GET /api/progress/:taskid`
- `GET /api/version`、`PUT /api/program`

### 设置与日志（手写路由）

- `/api/settings` 及 update-check / auto-backup / log-level / proxy
- `GET /api/logs`

### 健康检查

- `GET /healthz` — 不鉴权；进程存活 + Docker `Ping`（不可达 503）

### 数据目录

默认根路径 `/data`，可用环境变量 `DATA_DIR` 覆盖。备份/图标/设置/任务进度均相对该根目录；亦支持 `BACKUP_DIR`、`APP_SETTINGS_PATH`、`TASK_PROGRESS_PATH` 单独覆盖。
