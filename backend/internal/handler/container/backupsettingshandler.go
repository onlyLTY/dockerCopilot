package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func BackupSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewBackupSettingsLogic(r.Context(), svcCtx)
		resp, err := l.BackupSettings()
		writeLogicResp(w, r, resp, err)
	}
}
