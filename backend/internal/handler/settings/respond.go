package settings

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func writeLogicResp(w http.ResponseWriter, r *http.Request, resp *types.Resp, err error) {
	if err != nil {
		if resp != nil {
			httpx.OkJsonCtx(r.Context(), w, resp)
			return
		}
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	httpx.OkJsonCtx(r.Context(), w, resp)
}
