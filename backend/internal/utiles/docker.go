package utiles

import "github.com/onlyLTY/dockerCopilot/internal/svc"

// requireDocker 供本包内 Docker 操作使用；失败返回统一的 503 业务错误。
func requireDocker(ctx *svc.ServiceContext) error {
	return ctx.RequireDocker()
}
