package handler

import (
	"net/http"

	auth "github.com/onlyLTY/dockerCopilot/internal/handler/auth"
	container "github.com/onlyLTY/dockerCopilot/internal/handler/container"
	progress "github.com/onlyLTY/dockerCopilot/internal/handler/progress"
	"github.com/onlyLTY/dockerCopilot/internal/middleware"
	"github.com/onlyLTY/dockerCopilot/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterManualHandlers 注册手写路由与全局中间件。
//
// goctl 重新生成 routes.go 后，请确认 RegisterHandlers 仍会调用本函数；
// 本文件内的路由不要合并回 routes.go，避免下次生成被覆盖。
//
// 包含：
//   - 全局 SecurityHeaders
//   - POST /api/auth 登录限流
//   - GET /api/progress/list（批量任务进度）
//   - /api/settings* 与 /api/logs
func RegisterManualHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.Use(middleware.SecurityHeaders)

	loginLimit := middleware.NewLoginRateLimit()
	server.AddRoutes(
		rest.WithMiddleware(loginLimit.Handle,
			rest.Route{
				Method:  http.MethodPost,
				Path:    "/auth",
				Handler: auth.LoginHandler(serverCtx),
			},
		),
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/progress/list",
				Handler: progress.ListProgressHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/settings",
				Handler: GetAppSettingsHandler,
			},
			{
				Method:  http.MethodPut,
				Path:    "/settings",
				Handler: UpdateAppSettingsHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/settings/update-check",
				Handler: container.GetUpdateSettingsHandler,
			},
			{
				Method:  http.MethodPut,
				Path:    "/settings/update-check",
				Handler: container.UpdateUpdateSettingsHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/settings/auto-backup",
				Handler: container.GetAutoBackupSettingsHandler,
			},
			{
				Method:  http.MethodPut,
				Path:    "/settings/auto-backup",
				Handler: container.UpdateAutoBackupSettingsHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/settings/log-level",
				Handler: GetLogLevelHandler,
			},
			{
				Method:  http.MethodPut,
				Path:    "/settings/log-level",
				Handler: UpdateLogLevelHandler,
			},
			{
				Method:  http.MethodGet,
				Path:    "/settings/proxy",
				Handler: GetProxySettingsHandler,
			},
			{
				Method:  http.MethodPut,
				Path:    "/settings/proxy",
				Handler: UpdateProxySettingsHandler,
			},
			{
				Method:  http.MethodGet,
				Path:    "/logs",
				Handler: GetLogsHandler,
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/api"),
	)
}
