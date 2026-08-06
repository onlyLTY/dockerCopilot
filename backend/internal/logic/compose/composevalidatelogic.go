package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeValidateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeValidateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeValidateLogic {
	return &ComposeValidateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeValidateLogic) ComposeValidate(req *types.ComposeProjectValidateReq) (*types.Resp, error) {
	return NewFilesLogic(l.ctx, l.svcCtx).Validate(req)
}
