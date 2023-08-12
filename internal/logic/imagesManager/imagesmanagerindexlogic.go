package imagesManager

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
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

func (l *ImagesManagerIndexLogic) ImagesManagerIndex() error {
	// todo: add your logic here and delete this line

	return nil
}
