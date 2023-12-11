package version

import (
	"net/http"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/logic/version"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func CheckForUpdatesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := version.NewCheckForUpdatesLogic(r.Context(), svcCtx)
		resp, err := l.CheckForUpdates()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
