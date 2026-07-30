package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateIgnoreHandler(svcCtx *svc.ServiceContext, ignored bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContainerUpdateIgnoreReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := container.NewUpdateIgnoreLogic(r.Context(), svcCtx).Set(&req, ignored)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
