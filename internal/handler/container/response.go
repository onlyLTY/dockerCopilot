package container

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func writeContainerBadRequest(w http.ResponseWriter) {
	httpx.WriteJson(w, http.StatusBadRequest, types.Resp{
		Code: http.StatusBadRequest,
		Msg:  "请求参数错误",
		Data: map[string]interface{}{},
	})
}

func writeContainerResponse(w http.ResponseWriter, resp *types.Resp) {
	if resp == nil || resp.Code < 100 || resp.Code > 599 {
		httpx.WriteJson(w, http.StatusInternalServerError, types.Resp{
			Code: http.StatusInternalServerError,
			Msg:  "内部服务器错误",
			Data: map[string]interface{}{},
		})
		return
	}
	httpx.WriteJson(w, resp.Code, resp)
}
