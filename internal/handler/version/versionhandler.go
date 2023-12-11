package version

import (
	"net/http"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/logic/version"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func VersionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := version.NewVersionLogic(r.Context(), svcCtx)
		resp, err := l.Version()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.WriteJson(w, resp.Code, resp)
		}
	}
}
