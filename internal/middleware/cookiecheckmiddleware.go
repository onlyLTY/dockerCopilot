package middleware

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

type CookieCheckMiddleware struct {
	Uuid string
}

func NewCookieCheckMiddleware(uuid string) *CookieCheckMiddleware {
	return &CookieCheckMiddleware{Uuid: uuid}
}

func (m *CookieCheckMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
		cookies, err := r.Cookie("device_verified")
		if err != nil {
			if r.Method == http.MethodPost {
				httpx.OkJson(w, types.MsgResp{Status: "login_expired"})
				return
			}
			w.Header().Set("Location", "/login")
			w.WriteHeader(301)
			return
		}
		token, err := validateJwtToken(m.Uuid, cookies.Value)
		if err != nil {
			if r.Method == http.MethodPost {
				httpx.OkJson(w, types.MsgResp{Status: "login_expired"})
				return
			}
			w.Header().Set("Location", "/login")
			w.WriteHeader(301)
			return
		}
		if !token {
			if r.Method == http.MethodPost {
				httpx.OkJson(w, types.MsgResp{Status: "login_expired"})
				return
			}
			w.Header().Set("Location", "/login")
			w.WriteHeader(301)
			return
		}
		next(w, r)
	}
}
func validateJwtToken(secretKey string, tokenString string) (bool, error) {
	// 解析 JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法是否为 HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// 返回用于验证签名的密钥
		return []byte(secretKey), nil
	})

	// 验证解析结果
	if err != nil {
		return false, fmt.Errorf("failed to parse JWT token: %v", err)
	}

	// 校验 token 是否有效
	if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return true, nil
	} else {
		return false, fmt.Errorf("invalid JWT token")
	}
}
