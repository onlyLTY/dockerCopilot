package version

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/logic/version"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateprogramHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := version.NewUpdateprogramLogic(r.Context(), svcCtx)
		resp, err := l.Updateprogram()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
