package container

import (
	"net/http"
	"strconv"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func BackupSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		retention, err := backupstore.GetRetention()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 200, "msg": "success", "data": map[string]int{"retention": retention}})
	}
}

func UpdateBackupSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, err := strconv.Atoi(r.URL.Query().Get("retention"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		retention, err := backupstore.SetRetention(value)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 200, "msg": "success", "data": map[string]int{"retention": retention}})
	}
}
