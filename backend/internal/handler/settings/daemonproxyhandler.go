package settings

import (
	"net/http"

	daemonlogic "github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ApplyDaemonProxyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req daemonlogic.DaemonProxyRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := daemonlogic.NewDaemonProxyLogic(r.Context(), svcCtx).Apply(&req)
		writeLogicResp(w, r, resp, err)
	}
}

func RestartDaemonHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := daemonlogic.NewDaemonProxyLogic(r.Context(), svcCtx).Restart()
		writeLogicResp(w, r, resp, err)
	}
}

func DaemonOperationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DaemonOperationReq
		if err := httpx.ParsePath(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := daemonlogic.NewDaemonProxyLogic(r.Context(), svcCtx).Operation(req.OperationID)
		writeLogicResp(w, r, resp, err)
	}
}
