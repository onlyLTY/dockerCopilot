package imagesManager

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNewImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNewImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNewImageLogic {
	return &GetNewImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNewImageLogic) GetNewImage(req *types.GetNewImageReq) (resp *types.MsgResp, err error) {
	// todo: add your logic here and delete this line

	return
}
