package progress

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProgressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProgressLogic {
	return &ListProgressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListProgressLogic) ListProgress() (*types.Resp, error) {
	return NewGetProgressLogic(l.ctx, l.svcCtx).ListProgress()
}
