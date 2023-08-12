package containersManager

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopContainerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStopContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopContainerLogic {
	return &StopContainerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StopContainerLogic) StopContainer(req *types.StopContainerReq) (resp *types.MsgResp, err error) {
	// todo: add your logic here and delete this line
	msg, err := utiles.StopContainer(l.svcCtx, req.Name)
	return &msg, err
}
