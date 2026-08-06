package logic

import (
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

// Fail 填充业务失败响应；Docker 不可用时统一 503。
func Fail(resp *types.Resp, err error, code int, msg string) (*types.Resp, error) {
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

// Ok 成功响应。
func Ok(resp *types.Resp, data interface{}) *types.Resp {
	if data == nil {
		data = map[string]interface{}{}
	}
	resp.Code, resp.Msg, resp.Data = 200, "success", data
	return resp
}

// OkMsg 成功并自定义 msg。
func OkMsg(resp *types.Resp, msg string, data interface{}) *types.Resp {
	if data == nil {
		data = map[string]interface{}{}
	}
	if msg == "" {
		msg = "success"
	}
	resp.Code, resp.Msg, resp.Data = 200, msg, data
	return resp
}

// Biz 仅业务码（HTTP 仍 200），err 可为 nil。
func Biz(code int, msg string, data interface{}) *types.Resp {
	if data == nil {
		data = map[string]interface{}{}
	}
	return &types.Resp{Code: code, Msg: msg, Data: data}
}
