package containersManager

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
