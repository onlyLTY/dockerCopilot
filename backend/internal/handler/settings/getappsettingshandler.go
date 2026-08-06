package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetAppSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetAppSettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetAppSettings()
		writeLogicResp(w, r, resp, err)
	}
}
