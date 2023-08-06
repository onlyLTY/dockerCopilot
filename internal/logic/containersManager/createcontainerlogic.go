package containersManager

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateContainerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateContainerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateContainerLogic {
	return &CreateContainerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateContainerLogic) CreateContainer(req *types.CreateContainerReq) (resp *types.MsgResp, err error) {
	msg, err := utiles.CreateContainer(l.svcCtx, req.OldName, req.NewName, req.ImageNameAndTag)
	return &msg, err
}
