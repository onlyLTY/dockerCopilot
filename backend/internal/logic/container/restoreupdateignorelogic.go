package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreUpdateIgnoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreUpdateIgnoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreUpdateIgnoreLogic {
	return &RestoreUpdateIgnoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestoreUpdateIgnoreLogic) RestoreUpdateIgnore(req *types.IdReq) (*types.Resp, error) {
	legacy := &types.ContainerUpdateIgnoreReq{}
	legacy.Id = req.Id
	return NewUpdateIgnoreLogic(l.ctx, l.svcCtx).Set(legacy, false)
}
