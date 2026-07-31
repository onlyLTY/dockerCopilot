package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeCleanupPreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeCleanupPreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeCleanupPreviewLogic {
	return &ComposeCleanupPreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComposeCleanupPreviewLogic) ComposeCleanupPreview(req *types.ComposeCleanupPreviewReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
