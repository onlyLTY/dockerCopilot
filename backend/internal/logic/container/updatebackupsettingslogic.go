package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateBackupSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateBackupSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBackupSettingsLogic {
	return &UpdateBackupSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateBackupSettingsLogic) UpdateBackupSettings() (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
