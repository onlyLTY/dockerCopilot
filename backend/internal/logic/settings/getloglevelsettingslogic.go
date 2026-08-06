package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLogLevelSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLogLevelSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLogLevelSettingsLogic {
	return &GetLogLevelSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetLogLevelSettingsLogic) GetLogLevelSettings() (*types.Resp, error) {
	return logic.Biz(200, "success", map[string]interface{}{
		"level":   settingstore.GetLogLevel(),
		"options": settingstore.LogLevelOptions(),
	}), nil
}
