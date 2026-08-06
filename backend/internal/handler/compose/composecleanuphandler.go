package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ComposeCleanupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeCleanupReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("cleanup", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewComposeCleanupLogic(r.Context(), svcCtx).ComposeCleanup(&req)
		audit("cleanup", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
