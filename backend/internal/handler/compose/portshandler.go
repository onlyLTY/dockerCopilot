package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func PortsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		l := compose.NewPortsLogic(r.Context(), svcCtx)
		resp, err := l.Ports()
		audit("ports_list", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
