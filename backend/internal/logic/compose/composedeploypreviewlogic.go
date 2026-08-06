package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeDeployPreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeDeployPreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeDeployPreviewLogic {
	return &ComposeDeployPreviewLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeDeployPreviewLogic) ComposeDeployPreview(req *types.ComposeDeployPreviewReq) (*types.Resp, error) {
	return NewActionsLogic(l.ctx, l.svcCtx).DeployPreview(req)
}
