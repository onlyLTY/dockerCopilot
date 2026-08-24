package progress

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/progress"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetProgressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, types.Resp{
				Code: http.StatusBadRequest, Msg: "请求参数错误", Data: map[string]interface{}{},
			})
			return
		}

		l := progress.NewGetProgressLogic(r.Context(), svcCtx)
		resp, _ := l.GetProgress(&req)
		httpx.WriteJson(w, resp.Code, resp)
	}
}
