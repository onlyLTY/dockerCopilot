package icons

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/icons"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IconDeleteReq
		if err := httpx.Parse(r, &req); err != nil {
			// 兼容仅 query 的历史调用
			req.ImageName = r.URL.Query().Get("imageName")
		}
		if req.ImageName == "" {
			req.ImageName = r.URL.Query().Get("imageName")
		}
		l := icons.NewDeleteLogic(r.Context(), svcCtx)
		resp, err := l.Delete(req.ImageName)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
