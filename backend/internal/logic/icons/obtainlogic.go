package icons

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ObtainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewObtainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ObtainLogic {
	return &ObtainLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ObtainLogic) Obtain(req *types.RemoveImageReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
