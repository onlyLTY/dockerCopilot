package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func CheckUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewCheckUpdateLogic(r.Context(), svcCtx)
		resp, err := l.CheckUpdate()
		writeLogicResp(w, r, resp, err)
	}
}
