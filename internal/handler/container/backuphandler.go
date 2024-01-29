package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/logic/container"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func BackupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewBackupLogic(r.Context(), svcCtx)
		resp, err := l.Backup()
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
