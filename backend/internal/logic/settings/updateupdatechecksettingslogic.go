package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUpdateCheckSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUpdateCheckSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUpdateCheckSettingsLogic {
	return &UpdateUpdateCheckSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateUpdateCheckSettingsLogic) UpdateUpdateCheckSettings(req *types.IntervalSettingsReq) (*types.Resp, error) {
	interval, err := settingstore.SetUpdateCheckInterval(req.Interval)
	if err != nil {
		return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
	}
	if err := l.svcCtx.RescheduleUpdateCron(settingstore.UpdateCheckCron(interval)); err != nil {
		return logic.Biz(200, "设置已保存，但重新调度定时任务失败："+err.Error(), map[string]interface{}{
			"interval": interval,
			"options":  settingstore.UpdateCheckOptions(),
			"warning":  err.Error(),
		}), nil
	}
	return logic.Biz(200, "success", map[string]interface{}{
		"interval": interval,
		"options":  settingstore.UpdateCheckOptions(),
	}), nil
}
