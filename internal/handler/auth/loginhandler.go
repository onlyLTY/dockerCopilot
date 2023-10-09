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
			httpx.Error(w, err)
			return
		}
		l := auth.NewLoginLogic(r.Context(), ctx)
		resp, err := l.Login(&req)
		if err != nil {
			var resp types.MsgResp
			resp.Status = "error"
			httpx.WriteJson(w, 401, resp)
			return
		}
		httpx.OkJson(w, resp)
	}
}
