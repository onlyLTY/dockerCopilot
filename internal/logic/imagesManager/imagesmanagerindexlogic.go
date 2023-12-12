package imagesManager

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

type ImagesManagerIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImagesManagerIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImagesManagerIndexLogic {
	return &ImagesManagerIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImagesManagerIndexLogic) ImagesManagerIndex() ([]types.Image, error) {
	list, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		return nil, err
	}
	return list, nil
}
