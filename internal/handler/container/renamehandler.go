package container

import (
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/container"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

func RenameHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContainerRenameReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := container.NewRenameLogic(r.Context(), svcCtx)
		resp, err := l.Rename(&req)
		if err != nil {
			httpx.WriteJson(w, resp.Code, resp)
		} else {
			httpx.WriteJson(w, resp.Code, resp)
		}
	}
}
