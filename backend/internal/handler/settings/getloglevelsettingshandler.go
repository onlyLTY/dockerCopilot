package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetLogLevelSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetLogLevelSettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetLogLevelSettings()
		writeLogicResp(w, r, resp, err)
	}
}
