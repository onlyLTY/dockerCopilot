package handler

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func webindexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewWebindexLogic(r.Context(), svcCtx)
		err := l.Webindex()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
