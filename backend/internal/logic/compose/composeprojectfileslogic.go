package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectFilesLogic {
	return &ComposeProjectFilesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectFilesLogic) ComposeProjectFiles(req *types.ComposeProjectIdReq) (*types.Resp, error) {
	return NewFilesLogic(l.ctx, l.svcCtx).List(req)
}
