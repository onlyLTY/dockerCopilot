package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAutoBackupSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAutoBackupSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAutoBackupSettingsLogic {
	return &GetAutoBackupSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAutoBackupSettingsLogic) GetAutoBackupSettings() (*types.Resp, error) {
	return logic.Biz(200, "success", map[string]interface{}{
		"interval": settingstore.GetAutoBackupInterval(),
		"options":  settingstore.AutoBackupOptions(),
	}), nil
}
