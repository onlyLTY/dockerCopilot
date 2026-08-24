package image

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RemoveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RemoveImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, types.Resp{
				Code: http.StatusBadRequest, Msg: "请求参数错误", Data: map[string]interface{}{},
			})
			return
		}

		l := image.NewRemoveLogic(r.Context(), svcCtx)
		resp, _ := l.Remove(&req)
		httpx.WriteJson(w, resp.Code, resp)
	}
}
