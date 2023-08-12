package containersManager

import (
	"encoding/json"
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/containersManager"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func StopContainerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.StopContainerReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := containersManager.NewStopContainerLogic(r.Context(), svcCtx)
		resp, err := l.StopContainer(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
