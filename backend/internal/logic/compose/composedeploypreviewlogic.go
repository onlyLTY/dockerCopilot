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
	return &ComposeDeployPreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComposeDeployPreviewLogic) ComposeDeployPreview(req *types.ComposeDeployPreviewReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
