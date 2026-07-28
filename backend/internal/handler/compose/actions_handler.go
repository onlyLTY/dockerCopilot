package compose

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeployPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeDeployPreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewActionsLogic(r.Context(), svcCtx).DeployPreview(&req)
		writeResponse(r, w, resp, err)
	}
}

func DeployHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeDeployReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewActionsLogic(r.Context(), svcCtx).Deploy(&req)
		writeResponse(r, w, resp, err)
	}
}

func CleanupPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeCleanupPreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewActionsLogic(r.Context(), svcCtx).CleanupPreview(&req)
		writeResponse(r, w, resp, err)
	}
}

func CleanupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ComposeCleanupReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewActionsLogic(r.Context(), svcCtx).Cleanup(&req)
		writeResponse(r, w, resp, err)
	}
}
