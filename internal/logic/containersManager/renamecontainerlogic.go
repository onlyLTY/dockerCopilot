package containersManager

import (
	"context"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/utiles"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RenameContainerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRenameContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RenameContainerLogic {
	return &RenameContainerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RenameContainerLogic) RenameContainer(req *types.RenameContainerReq) (resp *types.MsgResp, err error) {
	err = utiles.RenameContainer(l.svcCtx, req.OldName, req.NewName)
	return &types.MsgResp{}, err
}
