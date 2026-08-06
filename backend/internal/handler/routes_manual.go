package handler

import (
	"net/http"

	auth "github.com/onlyLTY/dockerCopilot/internal/handler/auth"
	"github.com/onlyLTY/dockerCopilot/internal/middleware"
	"github.com/onlyLTY/dockerCopilot/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterManualHandlers 手写补丁（goctl 无法表达或需特殊中间件的部分）。
//
// 白名单：
//   - 全局 SecurityHeaders
//   - POST /api/auth 登录 IP 限流（.api 已声明 Login，但 gen routes 不含限流，故在此注册并勿在 routes.go 再挂裸 Login）
//
// 业务 settings/logs/progress 等以 goctl 生成的 routes.go 为准。
// 重新 goctl 后：务必在 RegisterHandlers 开头调用本函数，并删除 gen 产出的无中间件 /auth 路由。
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
}
