package handler

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// WriteLogicResp 按 goctl 惯例写出 logic 返回值。
// - err == nil：HTTP 200 + resp
// - err != nil 且 resp 非空：HTTP 200 + resp（业务码在 body.code，兼容现有前端）
// - err != nil 且 resp 为空：走全局 ErrorHandler（errorx.CodeError → 统一 JSON）
//
// 约定：业务失败时 logic 应同时填好 resp.Code/Msg，并可选返回 errorx 错误供日志/链路使用。
func WriteLogicResp(w http.ResponseWriter, r *http.Request, resp *types.Resp, err error) {
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
