package compose

import (
	"errors"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
)

func TestClientMsg(t *testing.T) {
	if got := clientMsg(nil, "兜底"); got != "兜底" {
		t.Fatalf("nil: %q", got)
	}
	if got := clientMsg(errorx.NewCodeError(400, "Compose 文件不能为空"), "x"); got != "Compose 文件不能为空" {
		t.Fatalf("codeerror: %q", got)
	}
	if got := clientMsg(errors.New("存在高风险配置: 服务启用了 privileged"), "x"); got != "存在高风险配置: 服务启用了 privileged" {
		t.Fatalf("cjk safe: %q", got)
	}
	if got := clientMsg(errors.New("Error response from daemon: conflict"), "部署失败"); got != "部署失败" {
		t.Fatalf("daemon leak should fallback: %q", got)
	}
	if got := clientMsg(errors.New("open /data/config/x: permission denied"), "读取失败"); got != "读取失败" {
		t.Fatalf("path leak should fallback: %q", got)
	}
}
