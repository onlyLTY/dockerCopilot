package Login

import (
	"github.com/flosch/pongo2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/logic/Login"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"time"
)

func DoLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	t, _ := svcCtx.Template.FromFile("templates/login/login.html")
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DoLoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := Login.NewDoLoginLogic(r.Context(), svcCtx)
		err := l.DoLogin(&req)
		if err != nil {

			//if err != nil {
			//	logx.Error(err)
			//}
			execute, err := t.ExecuteBytes(pongo2.Context{"current_year": time.Now(), "error_message": err.Error()})
			if err != nil {
				logx.Error(err)
			}
			w.Write(execute)
			httpx.Ok(w)
		} else {
			nowtime := time.Now()
			token, err := getJwtToken(svcCtx.Jwtuuid, nowtime.Unix(), svcCtx.Config.AccessExpire, "check_success")
			if err != nil {
				execute, err := t.ExecuteBytes(pongo2.Context{"current_year": time.Now(), "error_message": err.Error()})
				if err != nil {
					logx.Error(err)
				}
				w.Write(execute)
				httpx.Ok(w)
				return
			}
			cookies := &http.Cookie{
				Name:    "device_verified",
				Value:   token,
				Expires: nowtime.Add(time.Duration(svcCtx.Config.AccessExpire) * time.Second),
			}
			w.Header().Set("Set-Cookie", cookies.String())
			w.Header().Set("Location", "/containersManager")
			w.WriteHeader(301)
		}
	}
}
func getJwtToken(secretKey string, iat, seconds int64, payload string) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims["payload"] = payload
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
