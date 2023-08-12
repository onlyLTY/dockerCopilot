package version

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/version"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := version.NewUpdateLogic(r.Context(), svcCtx)
		resp, err := l.Update()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
