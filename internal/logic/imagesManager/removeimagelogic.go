package imagesManager

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveImageLogic {
	return &RemoveImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveImageLogic) RemoveImage(req *types.RemoveImageReq) (resp *types.MsgResp, err error) {
	// todo: add your logic here and delete this line
	msg, err := utiles.RemoveImage(l.svcCtx, req.ImageNameAndTag, req.Force)
	return &msg, err
}
