package settings

import (
	"context"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAppSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppSettingsLogic {
	return &UpdateAppSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateAppSettingsLogic) UpdateAppSettings(body *types.UpdateAppSettingsPartial) (*types.Resp, error) {
	var warnings []string

	if body.UpdateCheckInterval != nil {
		interval, err := settingstore.SetUpdateCheckInterval(*body.UpdateCheckInterval)
		if err != nil {
			return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
		}
		if err := l.svcCtx.RescheduleUpdateCron(settingstore.UpdateCheckCron(interval)); err != nil {
			warnings = append(warnings, "更新检查定时任务重新调度失败："+err.Error())
		}
	}

	if body.AutoBackupInterval != nil {
		interval, err := settingstore.SetAutoBackupInterval(*body.AutoBackupInterval)
		if err != nil {
			return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
		}
		if err := l.svcCtx.RescheduleBackupCron(settingstore.AutoBackupCron(interval)); err != nil {
			warnings = append(warnings, "自动备份定时任务重新调度失败："+err.Error())
		}
	}

	if body.LogLevel != nil {
		level, err := settingstore.SetLogLevel(*body.LogLevel)
		if err != nil {
			return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
		}
		logx.SetLevel(LogLevel(level))
	}

		if body.Retention != nil {
			if _, err := backupstore.SetRetention(*body.Retention); err != nil {
				return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
			}
		}

		if body.HubURLs != nil {
			if _, err := settingstore.SetHubURLs(*body.HubURLs); err != nil {
				return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
			}
		}

		if body.Proxy != nil {
			proxy := settingstore.ProxySettings{
				GithubProxy: body.Proxy.GithubProxy,
				HTTPProxy:   body.Proxy.HTTPProxy,
				HTTPSProxy:  body.Proxy.HTTPSProxy,
				NoProxy:     body.Proxy.NoProxy,
			}
			if _, err := settingstore.SetProxySettings(proxy); err != nil {
				return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
			}
		}

		msg := "success"
		if body.Proxy != nil {
			msg = "设置已保存，代理设置将在重启服务后生效"
		}
		if len(warnings) > 0 {
			msg = strings.Join(append([]string{msg}, warnings...), "；")
		}

		data := SnapshotAppSettings()
		if len(warnings) > 0 {
			return logic.Biz(200, msg, map[string]interface{}{
				"updateCheck": data.UpdateCheck,
				"autoBackup":  data.AutoBackup,
				"logLevel":    data.LogLevel,
				"retention":   data.Retention,
				"hubUrls":     data.HubURLs,
				"proxy":       data.Proxy,
				"warnings":    warnings,
			}), nil
		}
		return logic.Biz(200, msg, data), nil
	}
