package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeployPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return actionRequestHandler(svcCtx, "deploy_preview", func(l *compose.ActionsLogic, req *types.ComposeDeployPreviewReq) (*types.Resp, error) {
		return l.DeployPreview(req)
	})
}

func DeployHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return actionRequestHandler(svcCtx, "deploy", func(l *compose.ActionsLogic, req *types.ComposeDeployReq) (*types.Resp, error) {
		return l.Deploy(req)
	})
}

func CleanupPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return actionRequestHandler(svcCtx, "cleanup_preview", func(l *compose.ActionsLogic, req *types.ComposeCleanupPreviewReq) (*types.Resp, error) {
		return l.CleanupPreview(req)
	})
}

func CleanupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return actionRequestHandler(svcCtx, "cleanup", func(l *compose.ActionsLogic, req *types.ComposeCleanupReq) (*types.Resp, error) {
		return l.Cleanup(req)
	})
}

func actionRequestHandler[T any](svcCtx *svc.ServiceContext, operation string, fn func(*compose.ActionsLogic, *T) (*types.Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req T
		if err := httpx.Parse(r, &req); err != nil {
			auditError(operation, r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := fn(compose.NewActionsLogic(r.Context(), svcCtx), &req)
		audit(operation, r, resp, started)
		writeResponse(r, w, resp, err)
	}
}
