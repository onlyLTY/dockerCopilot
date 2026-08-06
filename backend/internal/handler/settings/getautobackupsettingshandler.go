package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetAutoBackupSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetAutoBackupSettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetAutoBackupSettings()
		writeLogicResp(w, r, resp, err)
	}
}
