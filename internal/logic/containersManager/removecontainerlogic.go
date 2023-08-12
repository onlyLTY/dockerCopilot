package containersManager

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
