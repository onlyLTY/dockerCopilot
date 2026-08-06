package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectFileLogic {
	return &ComposeProjectFileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectFileLogic) ComposeProjectFile(req *types.ComposeProjectFileReq) (*types.Resp, error) {
	return NewFilesLogic(l.ctx, l.svcCtx).Read(req)
}
