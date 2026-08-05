package container

import (
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

// fail 填充业务失败响应；Docker 不可用时统一 503。
func fail(resp *types.Resp, err error, code int, msg string) (*types.Resp, error) {
	if errorx.IsDockerUnavailable(err) {
		resp.Code = errorx.CodeDockerUnavailable
		resp.Msg = "Docker 服务不可用"
	} else {
		resp.Code = code
		resp.Msg = msg
	}
	resp.Data = map[string]interface{}{}
	return resp, err
}

func ok(resp *types.Resp, data interface{}) *types.Resp {
	if data == nil {
		data = map[string]interface{}{}
	}
	resp.Code, resp.Msg, resp.Data = 200, "success", data
	return resp
}
