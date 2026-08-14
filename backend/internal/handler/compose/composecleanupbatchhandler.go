package compose

import (
	"net/http"
	"time"

	composeLogic "github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ComposeCleanupBatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeCleanupBatchReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("cleanup_batch", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := composeLogic.NewComposeCleanupBatchLogic(r.Context(), svcCtx).Submit(&req)
		audit("cleanup_batch", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
