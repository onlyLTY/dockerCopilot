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
//   - POST /api/images/import 镜像导入：需要单独放宽请求体大小与超时（gen routes 无法表达）
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

	// 镜像导入：大请求体 + 长连接，单路由策略。MaxBytesHandler 会拒绝超过上限的
	// Content-Length；无 Content-Length 的 chunked 上传再由 handler 内的 MaxBytesReader 兜底。
	importLimit := serverCtx.Config.ImageImportLimit()
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/images/import",
				Handler: ImportImagesHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithTimeout(serverCtx.Config.ImageImportTimeout()),
		rest.WithMaxBytes(importLimit),
		rest.WithPrefix("/api"),
	)
}
