package version

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/version"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func VersionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.VersionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, types.Resp{
				Code: http.StatusBadRequest, Msg: "请求参数错误", Data: map[string]interface{}{},
			})
			return
		}

		l := version.NewVersionLogic(r.Context(), svcCtx)
		resp, _ := l.Version(&req)
		httpx.WriteJson(w, resp.Code, resp)
	}
}
