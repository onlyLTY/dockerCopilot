package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContainerUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			writeContainerBadRequest(w)
			return
		}

		l := container.NewUpdateLogic(r.Context(), svcCtx)
		resp, _ := l.Update(&req)
		writeContainerResponse(w, resp)
	}
}
