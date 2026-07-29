package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/onlyLTY/dockerCopilot/internal/logstore"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200, "msg": "success",
		"data": map[string]interface{}{
			"level":   settingstore.GetLogLevel(),
			"options": settingstore.LogLevelOptions(),
		},
	})
}

func UpdateLogLevelHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	level, err := settingstore.SetLogLevel(body.Level)
	if err != nil {
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 400, "msg": err.Error(), "data": map[string]interface{}{}})
		return
	}
	logx.SetLevel(logLevel(level))
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200, "msg": "success",
		"data": map[string]interface{}{"level": level, "options": settingstore.LogLevelOptions()},
	})
}

func GetLogsHandler(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil && r.URL.Query().Get("limit") != "" {
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	entries, err := logstore.ReadRecent(limit)
	if err != nil {
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 200, "msg": "success", "data": map[string]interface{}{"entries": entries}})
}

// logLevel 将设置项映射为 go-zero 的日志级别。go-zero 只有
// Debug < Info < Error 三档（没有独立的 warn），SetLevel 设置的是
// “写入的最低级别”。因此选 info 时，error 一定会被写入；选 error 时
// 只写 error。warn 没有对应档位，按 info 处理，避免误伤普通日志。
func logLevel(level string) uint32 {
	switch level {
	case "debug":
		return logx.DebugLevel
	case "error":
		return logx.ErrorLevel
	default: // info、warn 都按 info 处理
		return logx.InfoLevel
	}
}
