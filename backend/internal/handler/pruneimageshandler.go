package handler

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func pruneImagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ImagePruneReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewPruneImagesLogic(r.Context(), svcCtx)
		resp, err := l.PruneImages(&req)
		WriteLogicResp(w, r, resp, err)
	}
}
