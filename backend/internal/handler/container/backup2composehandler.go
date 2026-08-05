package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func Backup2composeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewBackup2composeLogic(r.Context(), svcCtx)
		resp, err := l.Backup2compose()
			writeLogicResp(w, r, resp, err)
	}
}
