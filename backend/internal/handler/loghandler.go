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

func logLevel(level string) uint32 {
	switch level {
	case "debug":
		return logx.DebugLevel
	case "error", "warn":
		return logx.ErrorLevel
	default:
		return logx.InfoLevel
	}
}
