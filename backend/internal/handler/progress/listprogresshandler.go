package progress

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/progress"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := progress.NewListProgressLogic(r.Context(), svcCtx)
		resp, err := l.ListProgress()
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
