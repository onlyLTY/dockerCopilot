package containersManager

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"

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
