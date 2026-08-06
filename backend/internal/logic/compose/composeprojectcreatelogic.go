package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectCreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectCreateLogic {
	return &ComposeProjectCreateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectCreateLogic) ComposeProjectCreate(req *types.ComposeProjectCreateReq) (*types.Resp, error) {
	return NewFilesLogic(l.ctx, l.svcCtx).Create(req)
}
