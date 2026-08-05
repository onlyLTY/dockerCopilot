package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// AppSettingsData 聚合设置快照（GET/PUT /api/settings）。
type AppSettingsData struct {
	UpdateCheck IntervalSettings          `json:"updateCheck"`
	AutoBackup  IntervalSettings          `json:"autoBackup"`
	LogLevel    LevelSettings             `json:"logLevel"`
	Retention   int                       `json:"retention"`
	Proxy       settingstore.ProxySettings `json:"proxy"`
}

type IntervalSettings struct {
	Interval string   `json:"interval"`
	Options  []string `json:"options"`
}

type LevelSettings struct {
	Level   string   `json:"level"`
	Options []string `json:"options"`
}

// UpdateAppSettingsReq 聚合写入请求；字段均可选，只更新出现的字段。
type UpdateAppSettingsReq struct {
	UpdateCheckInterval *string                     `json:"updateCheckInterval,optional"`
	AutoBackupInterval  *string                     `json:"autoBackupInterval,optional"`
	LogLevel            *string                     `json:"logLevel,optional"`
	Retention           *int                        `json:"retention,optional"`
	Proxy               *settingstore.ProxySettings `json:"proxy,optional"`
}

func snapshotAppSettings() AppSettingsData {
	return AppSettingsData{
		UpdateCheck: IntervalSettings{
			Interval: settingstore.GetUpdateCheckInterval(),
			Options:  settingstore.UpdateCheckOptions(),
		},
		AutoBackup: IntervalSettings{
			Interval: settingstore.GetAutoBackupInterval(),
			Options:  settingstore.AutoBackupOptions(),
		},
		LogLevel: LevelSettings{
			Level:   settingstore.GetLogLevel(),
			Options: settingstore.LogLevelOptions(),
		},
		Retention: settingstore.GetRetention(),
		Proxy:     settingstore.GetProxySettings(),
	}
}

// GetAppSettingsHandler 一次返回全部应用设置。
func GetAppSettingsHandler(w http.ResponseWriter, r *http.Request) {
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": snapshotAppSettings(),
	})
}

// UpdateAppSettingsHandler 一次写入多项设置，并执行各自副作用。
func UpdateAppSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body UpdateAppSettingsReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
				"code": 400, "msg": "请求体无效", "data": map[string]interface{}{},
			})
			return
		}

		var warnings []string

		if body.UpdateCheckInterval != nil {
			interval, err := settingstore.SetUpdateCheckInterval(*body.UpdateCheckInterval)
			if err != nil {
				httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
					"code": 400, "msg": err.Error(), "data": map[string]interface{}{},
				})
				return
			}
			if err := svcCtx.RescheduleUpdateCron(settingstore.UpdateCheckCron(interval)); err != nil {
				warnings = append(warnings, "更新检查定时任务重新调度失败："+err.Error())
			}
		}

		if body.AutoBackupInterval != nil {
			interval, err := settingstore.SetAutoBackupInterval(*body.AutoBackupInterval)
			if err != nil {
				httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
					"code": 400, "msg": err.Error(), "data": map[string]interface{}{},
				})
				return
			}
			if err := svcCtx.RescheduleBackupCron(settingstore.AutoBackupCron(interval)); err != nil {
				warnings = append(warnings, "自动备份定时任务重新调度失败："+err.Error())
			}
		}

		if body.LogLevel != nil {
			level, err := settingstore.SetLogLevel(*body.LogLevel)
			if err != nil {
				httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
					"code": 400, "msg": err.Error(), "data": map[string]interface{}{},
				})
				return
			}
			logx.SetLevel(logLevel(level))
		}

		if body.Retention != nil {
			if _, err := backupstore.SetRetention(*body.Retention); err != nil {
				httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
					"code": 400, "msg": err.Error(), "data": map[string]interface{}{},
				})
				return
			}
		}

		if body.Proxy != nil {
			if _, err := settingstore.SetProxySettings(*body.Proxy); err != nil {
				httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
					"code": 400, "msg": err.Error(), "data": map[string]interface{}{},
				})
				return
			}
		}

		msg := "success"
		if body.Proxy != nil {
			msg = "设置已保存，代理设置将在重启服务后生效"
		}
		if len(warnings) > 0 {
			msg = strings.Join(append([]string{msg}, warnings...), "；")
		}

		data := snapshotAppSettings()
		resp := map[string]interface{}{
			"code": 200,
			"msg":  msg,
			"data": data,
		}
		if len(warnings) > 0 {
			resp["data"] = map[string]interface{}{
				"updateCheck": data.UpdateCheck,
				"autoBackup":  data.AutoBackup,
				"logLevel":    data.LogLevel,
				"retention":   data.Retention,
				"proxy":       data.Proxy,
				"warnings":    warnings,
			}
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
