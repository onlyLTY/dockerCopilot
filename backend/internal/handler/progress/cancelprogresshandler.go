package progress

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/progress"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CancelProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelProgressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := progress.NewDeleteProgressLogic(r.Context(), svcCtx).CancelProgress(&req)
		if err != nil {
			if resp != nil {
				httpx.WriteJson(w, resp.Code, resp)
				return
			}
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
