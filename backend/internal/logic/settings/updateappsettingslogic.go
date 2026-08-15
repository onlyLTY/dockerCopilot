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
	patch := settingstore.AppSettingsPatch{
		UpdateCheckInterval: body.UpdateCheckInterval,
		AutoBackupInterval:  body.AutoBackupInterval,
		LogLevel:            body.LogLevel,
		Retention:           body.Retention,
		PullTimeoutSec:      body.PullTimeoutSec,
		HubURLs:             body.HubURLs,
	}
	if body.Proxy != nil {
		patch.Proxy = &settingstore.ProxySettings{GithubProxy: body.Proxy.GithubProxy, HTTPProxy: body.Proxy.HTTPProxy, HTTPSProxy: body.Proxy.HTTPSProxy, NoProxy: body.Proxy.NoProxy}
	}
	if body.DaemonProxyDraft != nil {
		patch.DaemonProxyDraft = &settingstore.DaemonProxyDraft{HTTPProxy: body.DaemonProxyDraft.HTTPProxy, HTTPSProxy: body.DaemonProxyDraft.HTTPSProxy, NoProxy: body.DaemonProxyDraft.NoProxy}
	}
	before := settingstore.Snapshot()
	stored, err := settingstore.ApplyPatch(patch)
	if err != nil {
		return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
	}
	var warnings []string
	if body.UpdateCheckInterval != nil {
		if err := l.svcCtx.RescheduleUpdateCron(settingstore.UpdateCheckCron(stored.UpdateCheckInterval)); err != nil {
			warnings = append(warnings, "更新检查定时任务重新调度失败："+err.Error())
		}
	}
	if body.AutoBackupInterval != nil {
		if err := l.svcCtx.RescheduleBackupCron(settingstore.AutoBackupCron(stored.AutoBackupInterval)); err != nil {
			warnings = append(warnings, "自动备份定时任务重新调度失败："+err.Error())
		}
	}
	if body.LogLevel != nil {
		logx.SetLevel(LogLevel(stored.LogLevel))
	}
	if body.Retention != nil {
		if err := backupstore.Retain(stored.Retention); err != nil {
			warnings = append(warnings, "清理旧备份失败："+err.Error())
		}
	}
	msg := "success"
	if body.Proxy != nil && body.DaemonProxyDraft != nil {
		msg = "设置已保存；Copilot 代理需重启服务生效，daemon 代理草稿需单独确认覆写"
	} else if body.Proxy != nil {
		msg = "设置已保存，代理设置将在重启服务后生效"
	} else if body.DaemonProxyDraft != nil {
		msg = "daemon 代理草稿已保存，尚未覆写宿主机配置"
	}
	if len(warnings) > 0 {
		msg = strings.Join(append([]string{msg}, warnings...), "；")
	}
	data := SnapshotAppSettings()
	if before.LogLevel != stored.LogLevel {
		logx.Infof("日志级别已更新为 %s", stored.LogLevel)
	}
	return logic.Biz(200, msg, data), nil
}
