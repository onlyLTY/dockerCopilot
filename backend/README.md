# dockerCopilot Backend Service

dockerCopilot 后端服务基于 [go-zero](https://github.com/zeromicro/go-zero) 框架开发，遵循 **Spec-First（教科书 A）** 架构：

```text
改 dockercopilot.api
  → goctl api go -api dockercopilot.api -dir . --style gozero
  → 只实现 / 调整 internal/logic/**
  → routes.go 以 goctl 生成为准（见下方「生成后补丁」）
```

## 目录架构

```
backend/
├── dockercopilot.api        # API 契约唯一真相 (Spec-First)
├── dockercopilot.go         # 服务入口：/healthz、静态资源、优雅退出
├── etc/
│   └── dockerCopilot.yaml   # 配置文件
├── internal/
│   ├── config/              # 配置结构体与版本解析
│   ├── errorx/              # 统一 CodeError
│   ├── handler/             # HTTP Handler + routes.go（goctl）+ routes_manual.go
│   ├── logic/               # 业务逻辑（人维护；goctl 仅生成不存在的 stub）
│   ├── middleware/          # 登录限流、安全响应头
│   ├── svc/                 # ServiceContext、任务进度、定时任务
│   ├── types/               # 请求/响应（goctl 生成 + 少量手写 partial）
│   └── utiles/              # Docker/Compose/备份等工具与子包
└── README.md
```

## 相关计划

- 教科书 A 改造计划与验收：[`docs/plan-gozero-textbook-A.md`](../docs/plan-gozero-textbook-A.md)

## 核心开发规范

1. **Spec-First**：API 变更**只**先改 `dockercopilot.api`，再执行：

   ```bash
   goctl api validate -api dockercopilot.api
   goctl api go -api dockercopilot.api -dir . --style gozero
   go test ./...
   ```

   **必须使用 `--style gozero`**（与现有小写连写文件名一致）。不要用 `go_zero`（snake_case 会生成双份文件）。

2. **生成后补丁（强制）**
   - `RegisterHandlers` **开头**调用 `RegisterManualHandlers`（安全头 + 登录限流版 `POST /api/auth`）。
   - **删除** goctl 生成的无中间件 `POST /api/auth`（避免与限流路由重复）。
   - 已存在的 logic/handler **不会被 goctl 覆盖**；新 stub 必须填满或 thin 转调，**禁止**提交 `todo: add your logic`。
   - `/healthz`、前端静态、图标 FileServer 仍在 `dockercopilot.go`。

3. **Handler → Logic**
   - 薄 Handler：`Parse` → `NewXxxLogic` → `writeLogicResp` / `WriteLogicResp`。
   - Compose 复杂域：goctl 入口 thin logic **转调** `ActionsLogic` / `FilesLogic`（不拆碎算法）。
   - Create 项目支持 JSON **或** form；icons 上传 multipart 在 handler 解析后交 logic。
   - settings 聚合 PUT 使用 `types.UpdateAppSettingsPartial`（指针字段，区分未传）。

4. **`routes_manual.go` 白名单**（仅此）
   - 全局 `SecurityHeaders`
   - `POST /api/auth` + 登录 IP 限流
   - **禁止**把 settings/logs 等业务长期只挂在 manual 而不进 `.api`

5. **错误处理**：业务错误用 `internal/errorx.CodeError`；Logic 填好 `resp.Code/Msg`；handler 以 HTTP 200 + body 为主（icons 上传等历史契约除外）。

6. **Docker 客户端**：`svcCtx.RequireDocker()` / `requireDocker`；不可用业务码 **503**。

7. **日志**：`SetupLog` 按天轮转，`KeepDays=7`。

8. **编译与防回潮**：`go test ./...`（含 `routes_guard_test`：强制 `RegisterManualHandlers`、禁止 logic todo stub）与 `go build ./...`。

## 启动与运行

本地推荐固定配置（`etc/dockerCopilot.local.yaml` 已 gitignore，内含 `DataDir`）：

```bash
# 首次：cp etc/dockerCopilot.yaml etc/dockerCopilot.local.yaml
# 编辑：AccessSecret、DataDir: ../docker/test/data、Compose.ScanPaths
go run dockercopilot.go -f etc/dockerCopilot.local.yaml
```

或主配置 + 环境变量：

```bash
export secretKey=test123456
# 可选：export DATA_DIR=... 覆盖 yaml DataDir
go run dockercopilot.go -f etc/dockerCopilot.yaml
go build ./...
go test ./...
```

- `etc/dockerCopilot.yaml`：`Auth.AccessSecret: ${secretKey}`（启动强制：至少 8 位且不能为纯数字）；`Timeout` 默认 10 分钟；`AccessExpire` 默认 7 天；`DataDir` 可省略（容器挂 `/data`）
- `etc/dockerCopilot.local.yaml`：本机开发；密钥、`DataDir`、`Compose.ScanPaths` 写死；`Compose.BackupDir` 可省略
- **数据根**：`DATA_DIR` 环境变量 > yaml `DataDir` > 默认 `/data`（Windows 未配置时常为 `E:\data`）
- 进程退出：`proc` WrapUp 刷任务进度，Shutdown 停 cron 并关闭 Docker 客户端

## 核心接口说明

### 认证

- `POST /api/auth` — 登录（含 IP 失败限流，状态持久化到 `loginAttempts.json`，见 `routes_manual`）

### 容器

- `GET /api/containers`
- `POST /api/containers/check-update`
- `POST /api/container/:id/start|stop|restart|rename|update`
- `POST|DELETE /api/container/:id/update-ignore`
- 备份：`/api/container/backup*`、`listBackups`、`backups/restore` 等

### Compose

- `GET|POST /api/compose/projects`
- `POST .../deploy`、`deploy/preview`、cleanup、`POST /api/compose/validate`
- Handler → thin logic → `ActionsLogic` / `FilesLogic`

### 镜像 / 端口 / 图标 / 进度 / 版本

- `GET /api/images`、`POST /api/images/prune`、`DELETE /api/image/:id`
- `GET /api/ports`
- `/api/icons`
- `GET /api/progress/list`、`GET /api/progress/:taskid`
- `GET /api/version`、`PUT /api/program`

### 设置与日志（已在 `.api` + gen routes）

- `/api/settings` 及 update-check / auto-backup / log-level / proxy
- `GET /api/logs`

### 健康检查

- `GET /healthz` — 不鉴权；进程存活 + Docker `Ping`（不可达 503）

### 数据目录

数据根优先级：环境变量 `DATA_DIR` > yaml `DataDir` > 默认 `/data`。本地推荐在 `dockerCopilot.local.yaml` 写 `DataDir: ../docker/test/data`。其下：

| 路径 | 用途 |
|------|------|
| `{DataDir}/config/appSettings.json` | 应用设置（含 Docker 加速源 `hubUrls`） |
| `{DataDir}/backups/` | **容器备份页** / 自动备份（JSON、YAML） |
| `{DataDir}/icon/` | 自定义图标 |
| `{DataDir}/config/taskProgress.json` | 任务进度 |

`Compose.BackupDir`（yaml）可省略，默认 `{DataDir}/backups/compose-projects`（或 `BACKUP_DIR/compose-projects`），仅用于手动备份 Compose 项目：支持 `compose.yaml`、`compose.yml`、`docker-compose.yaml`、`docker-compose.yml`，输出文件名为 `<container_name>_compose.yaml`；存在 `.env` 和 `config.yaml` 时会在 `{ProjectID}` 子目录中一并复制这两个文件。没有唯一 `container_name` 时使用 Compose 项目名。保存文件和清理项目不会自动创建 Compose 备份。它**不是**备份页目录。亦支持 `BACKUP_DIR`、`APP_SETTINGS_PATH`、`TASK_PROGRESS_PATH` 单独覆盖。
