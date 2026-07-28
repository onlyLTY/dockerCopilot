package container

import (
	"encoding/json"
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetUpdateSettingsHandler 返回当前更新检查频率与可选项。
func GetUpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200, "msg": "success",
		"data": map[string]interface{}{
			"interval": settingstore.GetUpdateCheckInterval(),
			"options":  settingstore.UpdateCheckOptions(),
		},
	})
}

// UpdateUpdateSettingsHandler 更新检查频率设置，并按新频率重新调度定时任务。
func UpdateUpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Interval string `json:"interval"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		interval, err := settingstore.SetUpdateCheckInterval(body.Interval)
		if err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 400, "msg": err.Error(), "data": map[string]interface{}{}})
			return
		}
		if err := svcCtx.RescheduleUpdateCron(settingstore.UpdateCheckCron(interval)); err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 500, "msg": "重新调度定时任务失败：" + err.Error(), "data": map[string]interface{}{}})
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
			"code": 200, "msg": "success",
			"data": map[string]interface{}{
				"interval": interval,
				"options":  settingstore.UpdateCheckOptions(),
			},
		})
	}
}

// GetAutoBackupSettingsHandler 返回当前自动备份频率与可选项。
func GetAutoBackupSettingsHandler(w http.ResponseWriter, r *http.Request) {
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200, "msg": "success",
		"data": map[string]interface{}{
			"interval": settingstore.GetAutoBackupInterval(),
			"options":  settingstore.AutoBackupOptions(),
		},
	})
}

// UpdateAutoBackupSettingsHandler 更新自动备份频率设置，并按新频率重新调度定时任务。
func UpdateAutoBackupSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Interval string `json:"interval"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		interval, err := settingstore.SetAutoBackupInterval(body.Interval)
		if err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 400, "msg": err.Error(), "data": map[string]interface{}{}})
			return
		}
		if err := svcCtx.RescheduleBackupCron(settingstore.AutoBackupCron(interval)); err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 500, "msg": "重新调度定时任务失败：" + err.Error(), "data": map[string]interface{}{}})
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
			"code": 200, "msg": "success",
			"data": map[string]interface{}{
				"interval": interval,
				"options":  settingstore.AutoBackupOptions(),
			},
		})
	}
}
