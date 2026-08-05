// Package errorx 提供业务错误类型，与 go-zero httpx.ErrorHandler 配合：
// handler 对 error 调用 httpx.ErrorCtx，由全局 SetErrorHandler 写成统一 JSON。
package errorx

import (
	"errors"
	"fmt"
)

// 常用业务码（与前端约定：HTTP 多为 200，body.code 表示业务结果）。
const (
	CodeBadRequest          = 400
	CodeUnauthorized        = 401
	CodeNotFound            = 404
	CodeConflict            = 409
	CodeDockerUnavailable   = 503
	CodeInternal            = 500
	CodeInternalUnhandled   = 50000
)

// CodeError 可安全返回给客户端的业务错误。
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// CodeErrorResponse 序列化形状（兼容旧调用）。
type CodeErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func NewCodeError(code int, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

func NewDefaultError(msg string) error {
	return NewCodeError(CodeInternalUnhandled, msg)
}

// ErrDockerUnavailable Docker Engine 客户端未初始化或不可用。
var ErrDockerUnavailable = NewCodeError(CodeDockerUnavailable, "Docker 服务不可用")

func (e *CodeError) Error() string {
	if e == nil {
		return ""
	}
	// 给 clientMsg / 日志用短文案；完整 Code 在字段里
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("code %d", e.Code)
}

func (e *CodeError) Data() *CodeErrorResponse {
	if e == nil {
		return &CodeErrorResponse{}
	}
	return &CodeErrorResponse{Code: e.Code, Msg: e.Msg}
}

// AsCodeError 从 error 链取出 *CodeError。
func AsCodeError(err error) (*CodeError, bool) {
	var ce *CodeError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}

// IsDockerUnavailable 是否为 Docker 不可用错误。
func IsDockerUnavailable(err error) bool {
	ce, ok := AsCodeError(err)
	return ok && ce.Code == CodeDockerUnavailable
}
