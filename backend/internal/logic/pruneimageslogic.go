package logic

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PruneImagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPruneImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PruneImagesLogic {
	return &PruneImagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PruneImagesLogic) PruneImages(req *types.ImagePruneReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
