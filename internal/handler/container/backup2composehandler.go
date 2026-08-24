package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func Backup2composeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewBackup2composeLogic(r.Context(), svcCtx)
		resp, _ := l.Backup2compose()
		writeContainerResponse(w, resp)
	}
}
