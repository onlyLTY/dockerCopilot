package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackupLogic {
	return &BackupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BackupLogic) Backup() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.BackupContainer(l.svcCtx)
	if err != nil {
		l.Errorf("备份容器失败: %v", err)
		return fail(resp, err, 500, "备份容器失败")
	}
	return ok(resp, nil), nil
}
