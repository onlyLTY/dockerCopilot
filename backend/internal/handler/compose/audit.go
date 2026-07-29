package compose

import (
	"net/http"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func audit(op string, r *http.Request, resp *types.Resp, started time.Time) {
	code := 500
	message := "nil response"
	if resp != nil {
		code = resp.Code
		message = resp.Msg
	}
	logx.Infof(
		"compose operation=%s method=%s path=%s code=%d message=%q duration_ms=%d",
		op,
		r.Method,
		r.URL.Path,
		code,
		message,
		time.Since(started).Milliseconds(),
	)
}

func auditError(op string, r *http.Request, err error, started time.Time) {
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}
	logx.Errorf(
		"compose operation=%s method=%s path=%s code=400 message=%q duration_ms=%d",
		op,
		r.Method,
		r.URL.Path,
		message,
		time.Since(started).Milliseconds(),
	)
}
