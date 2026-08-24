package auth

import (
	"github.com/onlyLTY/dockerCopilot/internal/logic/auth"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"strconv"
	"time"
)

func LoginHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientKey := loginClientKey(r)
		if allowed, retryAfter := authLimiter.allow(clientKey, time.Now()); !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Round(time.Second).Seconds())))
			httpx.WriteJson(w, http.StatusTooManyRequests, types.Resp{
				Code: http.StatusTooManyRequests, Msg: "登录失败次数过多，请稍后重试",
				Data: map[string]interface{}{},
			})
			return
		}
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
		authLimiter.record(clientKey, err == nil, time.Now())
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
			return
		}
		httpx.OkJson(w, resp)
	}
}
