package progress

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/logic/progress"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetProgressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := progress.NewGetProgressLogic(r.Context(), svcCtx)
		resp, err := l.GetProgress(&req)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
