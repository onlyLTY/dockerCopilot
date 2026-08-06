package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ComposeCleanupPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeCleanupPreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("cleanup_preview", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewComposeCleanupPreviewLogic(r.Context(), svcCtx).ComposeCleanupPreview(&req)
		audit("cleanup_preview", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
