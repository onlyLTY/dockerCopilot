package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type Backup2composeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackup2composeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Backup2composeLogic {
	return &Backup2composeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *Backup2composeLogic) Backup2compose() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.Backup2Compose(l.svcCtx)
	if err != nil {
		return nil, err
	}
	resp.Code = 200
	resp.Msg = "success"
	return
}
