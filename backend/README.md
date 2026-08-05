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

1. **Spec-First**：API 变更优先改 `dockercopilot.api`，再 `goctl api go -api dockercopilot.api -dir .` 生成框架代码。
2. **手写路由**：扩展路由与中间件在 `internal/handler/routes_manual.go`（`RegisterManualHandlers`）。`routes.go` 为 goctl 业务路由；重新生成后务必保留对 `RegisterManualHandlers` 的调用。
3. **错误处理**：对客户端返回可读业务文案（Compose 使用 `clientMsg`），底层错误只写日志。
4. **日志**：`SetupLog` 使用 go-zero file 模式，**按天轮转**（`Rotation=daily`），`KeepDays=7` 并压缩历史；不是无限追加单个文件。
5. **编译与验证**：`go test ./...` 与 `go build ./...`。

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

- `etc/dockerCopilot.yaml`：通用/容器用，`Auth.AccessSecret: ${secretKey}`
- `etc/dockerCopilot.local.yaml`：本机开发，AccessSecret 可写明文；`Compose.AllowHighRisk` 控制极高危部署门禁

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
- 文件读写、cleanup 等

### 镜像 / 端口 / 图标 / 进度 / 版本

- `GET /api/images`、`POST /api/images/prune`、`DELETE /api/image/:id`（JWT + `/api` 前缀）
- `GET /api/ports`
- `/api/icons`
- `GET /api/progress/list`、`GET /api/progress/:taskid`
- `GET /api/version`、`PUT /api/program`

### 设置与日志（手写路由）

- `/api/settings` 及 update-check / auto-backup / log-level / proxy
- `GET /api/logs`
