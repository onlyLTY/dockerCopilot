package compose

import (
	"strings"
	"unicode"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
)

// clientMsg 将内部 error 转为可展示给前端的短文案。
// - errorx.CodeError：直接用业务 Msg
// - 已是中文/业务校验类：原样返回（来自 compose_project 等）
// - 其余（路径、Docker 细节等）：用 fallback，完整错误只应写日志
func clientMsg(err error, fallback string) string {
	if err == nil {
		if fallback != "" {
			return fallback
		}
		return "操作失败"
	}
	if ce, ok := errorx.AsCodeError(err); ok {
		if msg := strings.TrimSpace(ce.Msg); msg != "" {
			return msg
		}
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		if fallback != "" {
			return fallback
		}
		return "操作失败"
	}
	// 兼容旧 Error() 格式 "Code: 400, Msg: xxx"（若仍有）
	if strings.HasPrefix(msg, "Code:") {
		if i := strings.Index(msg, "Msg:"); i >= 0 {
			inner := strings.TrimSpace(msg[i+4:])
			if inner != "" && looksClientSafe(inner) {
				return inner
			}
		}
	}
	if looksClientSafe(msg) {
		return msg
	}
	if fallback != "" {
		return fallback
	}
	return "操作失败"
}

func looksClientSafe(msg string) bool {
	lower := strings.ToLower(msg)
	// 典型底层泄露
	leaks := []string{
		"error response from daemon",
		"dial unix",
		"dial tcp",
		"permission denied",
		"no such file",
		"connect: ",
		"i/o timeout",
		"context deadline",
		"json:",
		"yaml:",
		"stack",
		"panic",
		"/var/",
		"/usr/",
		"/home/",
		"c:\\",
		"\\\\",
	}
	for _, p := range leaks {
		if strings.Contains(lower, p) {
			return false
		}
	}
	// 过长多半是命令输出/堆栈
	if len(msg) > 240 {
		return false
	}
	// 含绝对 Unix 路径
	if strings.Contains(msg, "/compose/") || strings.Contains(msg, "/data/") {
		return false
	}
	hasCJK := false
	for _, r := range msg {
		if unicode.Is(unicode.Han, r) {
			hasCJK = true
			break
		}
	}
	// 业务文案几乎都是中文；纯英文短句也允许少量已知前缀
	if hasCJK {
		return true
	}
	safeEN := []string{
		"compose",
		"project",
		"invalid",
		"not found",
		"conflict",
	}
	for _, p := range safeEN {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
