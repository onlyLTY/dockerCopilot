package container

import (
	"context"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/utiles"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBackupsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBackupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBackupsLogic {
	return &ListBackupsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBackupsLogic) ListBackups() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	backupList, err := utiles.BackupList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		return resp, err
	}
	resp.Msg = "success"
	resp.Code = 200
	resp.Data = backupList
	return resp, nil
}
