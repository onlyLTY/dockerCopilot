package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUpdateCheckSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUpdateCheckSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUpdateCheckSettingsLogic {
	return &GetUpdateCheckSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetUpdateCheckSettingsLogic) GetUpdateCheckSettings() (*types.Resp, error) {
	return logic.Biz(200, "success", map[string]interface{}{
		"interval": settingstore.GetUpdateCheckInterval(),
		"options":  settingstore.UpdateCheckOptions(),
	}), nil
}
