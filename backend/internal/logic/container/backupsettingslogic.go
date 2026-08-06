package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
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
	return &BackupSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *BackupSettingsLogic) BackupSettings() (*types.Resp, error) {
	resp := &types.Resp{}
	retention, err := backupstore.GetRetention()
	if err != nil {
		return fail(resp, err, 500, "读取备份保留设置失败")
	}
	return ok(resp, map[string]int{"retention": retention}), nil
}
