package containersManager

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartContainerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartContainerLogic {
	return &StartContainerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartContainerLogic) StartContainer(req *types.StartContainerReq) (resp *types.MsgResp, err error) {
	msg, err := utiles.StartContainer(l.svcCtx, req.Name)
	return &msg, err
}
