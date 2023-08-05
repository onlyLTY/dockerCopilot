package containersManager

import (
	"encoding/json"
	"net/http"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/logic/containersManager"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RenameContainerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RenameContainerReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := containersManager.NewRenameContainerLogic(r.Context(), svcCtx)
		resp, err := l.RenameContainer(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
