package containersManager

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveContainerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveContainerLogic {
	return &RemoveContainerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveContainerLogic) RemoveContainer(req *types.RemoveContainerReq) (resp *types.MsgResp, err error) {
	msg, err := utiles.RemoveContainer(l.svcCtx, req.Name)
	return &msg, err
}
