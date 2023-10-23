package container

import (
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/logic/container"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListBackupsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := container.NewListBackupsLogic(r.Context(), svcCtx)
		resp, err := l.ListBackups()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
