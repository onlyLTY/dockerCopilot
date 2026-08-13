package settings

import (
	"net/http"

	daemonlogic "github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetDaemonProxyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := daemonlogic.NewDaemonProxyLogic(r.Context(), svcCtx).Get()
		writeLogicResp(w, r, resp, err)
	}
}
