package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BackupSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackupSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackupSettingsLogic {
	return &BackupSettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BackupSettingsLogic) BackupSettings() (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
