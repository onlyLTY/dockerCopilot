package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetProxySettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetProxySettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetProxySettings()
		writeLogicResp(w, r, resp, err)
	}
}
