package containersManager

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type ContainersManagerIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewContainersManagerIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainersManagerIndexLogic {
	return &ContainersManagerIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainersManagerIndexLogic) ContainersManagerIndex() error {
	// todo: add your logic here and delete this line

	return nil
}
