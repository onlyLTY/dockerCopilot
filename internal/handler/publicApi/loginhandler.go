package publicApi

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/publicApi"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"time"
)

func LoginHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}
		l := publicApi.NewLoginLogic(r.Context(), ctx)
		secretKey, err := l.Login(&req)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(time.Hour * 720).Unix(),
		})

		tokenString, err := token.SignedString([]byte(secretKey))
		if err != nil {
			httpx.Error(w, errors.New("无法生成 token请重试"))
			return
		}
		httpx.OkJson(w, types.LoginResp{Token: tokenString})
	}
}
