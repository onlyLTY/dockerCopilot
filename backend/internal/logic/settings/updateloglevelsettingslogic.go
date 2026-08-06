package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogLevelSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogLevelSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogLevelSettingsLogic {
	return &UpdateLogLevelSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogLevelSettingsLogic) UpdateLogLevelSettings(req *types.LogLevelReq) (*types.Resp, error) {
	level, err := settingstore.SetLogLevel(req.Level)
	if err != nil {
		return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
	}
	logx.SetLevel(LogLevel(level))
	return logic.Biz(200, "success", map[string]interface{}{
		"level":   level,
		"options": settingstore.LogLevelOptions(),
	}), nil
}
