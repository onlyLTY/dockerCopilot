package containersManager

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/utiles"
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

func (l *ContainersManagerIndexLogic) ContainersManagerIndex() (*[]types.Container, error) {
	list, err := utiles.GetContainerList(l.svcCtx)
	if err != nil {
		return nil, err
	}
	utiles.CheckImageUpdate(l.svcCtx, list)
	return &list, nil
}
