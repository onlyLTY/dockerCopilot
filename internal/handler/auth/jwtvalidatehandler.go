package auth

import (
	"net/http"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/logic/auth"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func JwtValidateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := auth.NewJwtValidateLogic(r.Context(), svcCtx)
		resp, err := l.JwtValidate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
