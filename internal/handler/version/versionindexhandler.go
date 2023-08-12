package version

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/logic/version"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func VersionIndexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := version.NewVersionIndexLogic(r.Context(), svcCtx)
		err := l.VersionIndex()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
