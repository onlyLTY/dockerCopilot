# dockerCopilot Backend Service

dockerCopilot 后端服务基于 [go-zero](https://github.com/zeromicro/go-zero) 框架开发，遵循 Spec-First 规范架构设计。

## 目录架构

```
backend/
├── dockercopilot.api        # API 规范描述文件 (Spec-First)
├── dockercopilot.go         # 服务入口文件
├── etc/
│   └── dockercopilot.yaml   # 配置文件
├── internal/
│   ├── config/              # 配置结构体定义
│   ├── errorx/              # 统一 CodeError 异常与错误处理包
│   ├── handler/             # HTTP 路由与 Handler 层 (goctl 生成)
│   ├── logic/               # 业务逻辑代码实现层
│   ├── svc/                 # ServiceContext 服务上下文
│   ├── types/               # 请求/响应数据结构体定义 (goctl 生成)
│   └── utiles/              # 工具函数与辅助模块
└── README.md
```

## 核心开发规范

1. **Spec-First 规范**：所有 RESTful API 的变更必须优先修改 `dockercopilot.api`，并使用 `goctl api go -api dockercopilot.api -dir .` 生成框架代码。
2. **请求参数校验**：所有 Request 结构体必须在 `.api` 文件中明确配置 `validate` 校验规则标签（如 `validate:"required"`）。
3. **错误处理标准**：Logic 逻辑层统一使用 `internal/errorx` 抛出标准 CodeError（如 `errorx.NewCodeError(code, msg)`），严禁使用 raw `fmt.Errorf` 泄露底层错误。
4. **编译与验证**：每次变更后需按顺序执行 `go mod tidy` 与 `go build ./...` 确保代码可以通过编译。

## 启动与运行

```bash
# 启动服务
go run dockercopilot.go -f etc/dockercopilot.yaml

# 验证构建
go build ./...
```

## 核心接口说明

### 认证接口 (Auth)
- `POST /api/auth` - 用户登录鉴权

### 容器管理 (Container)
- `GET /api/containers` - 获取容器列表
- `POST /api/container/:id/start` - 启动容器
- `POST /api/container/:id/stop` - 停止容器
- `POST /api/container/:id/restart` - 重启容器
- `POST /api/container/:id/rename` - 容器重命名
- `POST /api/container/:id/update` - 升级/更新容器

### Compose 管理 (Compose)
- `GET /api/compose/projects` - 获取 Compose 项目列表
- `POST /api/compose/projects` - 创建 Compose 项目
- `GET /api/compose/projects/:id/files` - 查看项目文件
- `POST /api/compose/projects/:id/deploy` - 部署 Compose 项目

### 图标管理 (Icons)
- `POST /api/icons/` - 上传图标
- `GET /api/icons/` - 获取图标
- `DELETE /api/icons/` - 删除图标
