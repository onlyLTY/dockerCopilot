package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAppSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppSettingsLogic {
	return &GetAppSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAppSettingsLogic) GetAppSettings() (*types.Resp, error) {
	return logic.Biz(200, "success", SnapshotAppSettings()), nil
}
