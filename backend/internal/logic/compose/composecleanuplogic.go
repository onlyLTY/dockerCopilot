package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeCleanupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeCleanupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeCleanupLogic {
	return &ComposeCleanupLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeCleanupLogic) ComposeCleanup(req *types.ComposeCleanupReq) (*types.Resp, error) {
	return NewActionsLogic(l.ctx, l.svcCtx).Cleanup(req)
}
