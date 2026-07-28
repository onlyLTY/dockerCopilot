package image

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RemoveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RemoveImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := image.NewRemoveLogic(r.Context(), svcCtx).Remove(&req)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.WriteJson(w, resp.Code, resp)
	}
}

func PruneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ImagePruneReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := image.NewPruneLogic(r.Context(), svcCtx).Prune(&req)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.WriteJson(w, resp.Code, resp)
	}
}
