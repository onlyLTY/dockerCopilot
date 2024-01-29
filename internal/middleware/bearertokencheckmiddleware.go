package middleware

import (
	"github.com/golang-jwt/jwt"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"strings"
)

type BearerTokenCheckMiddleware struct {
	secretKey string
}

func NewBearerTokenCheckMiddleware(secretKey string) *BearerTokenCheckMiddleware {
	return &BearerTokenCheckMiddleware{secretKey: secretKey}
}

func (m *BearerTokenCheckMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp types.Resp
		authHeader := r.Header.Get("Authorization")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			resp.Code = 401
			resp.Msg = "Unauthorized"
			httpx.WriteJson(w, resp.Code, resp)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.secretKey), nil
		})
		if err != nil || !token.Valid {
			resp.Code = 401
			resp.Msg = "auth out of date"
			httpx.WriteJson(w, resp.Code, resp)
			return
		}

		next(w, r)
	}
}
