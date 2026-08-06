package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeDeployLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeDeployLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeDeployLogic {
	return &ComposeDeployLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeDeployLogic) ComposeDeploy(req *types.ComposeDeployReq) (*types.Resp, error) {
	return NewActionsLogic(l.ctx, l.svcCtx).Deploy(req)
}
