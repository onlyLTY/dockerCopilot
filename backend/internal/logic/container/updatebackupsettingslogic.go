package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
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

func (l *UpdateBackupSettingsLogic) UpdateBackupSettings(req *types.UpdateBackupSettingsReq) (*types.Resp, error) {
	resp := &types.Resp{}
	retention, err := backupstore.SetRetention(req.Retention)
	if err != nil {
		return fail(resp, err, 400, err.Error())
	}
	return ok(resp, map[string]int{"retention": retention}), nil
}
