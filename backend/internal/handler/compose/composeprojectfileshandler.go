package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/logic/compose"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ComposeProjectFilesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var req types.ComposeProjectIdReq
		if err := httpx.Parse(r, &req); err != nil {
			auditError("file_list", r, err, started)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := compose.NewComposeProjectFilesLogic(r.Context(), svcCtx).ComposeProjectFiles(&req)
		audit("file_list", r, resp, started)
		writeLogicResp(w, r, resp, err)
	}
}
