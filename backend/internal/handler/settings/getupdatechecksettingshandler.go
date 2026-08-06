package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetUpdateCheckSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetUpdateCheckSettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetUpdateCheckSettings()
		writeLogicResp(w, r, resp, err)
	}
}
