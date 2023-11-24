package auth

import (
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/auth"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

func LoginHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			var resp types.Resp
			resp.Code = 400
			resp.Msg = "错误的请求"
			httpx.WriteJson(w, 400, resp)
			return
		}
		l := auth.NewLoginLogic(r.Context(), ctx)
		resp, err := l.Login(&req)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJson(w, resp)
	}
}
