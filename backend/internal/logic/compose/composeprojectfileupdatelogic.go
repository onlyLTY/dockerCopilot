package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectFileUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectFileUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectFileUpdateLogic {
	return &ComposeProjectFileUpdateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectFileUpdateLogic) ComposeProjectFileUpdate(req *types.ComposeProjectFileUpdateReq) (*types.Resp, error) {
	return NewFilesLogic(l.ctx, l.svcCtx).Update(req)
}
