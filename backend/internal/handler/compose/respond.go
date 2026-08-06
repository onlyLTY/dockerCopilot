package compose

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// writeLogicResp 与根包 handler.WriteLogicResp 相同约定，供 compose 子包使用
// （避免 handler → handler/compose 循环依赖）。
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
