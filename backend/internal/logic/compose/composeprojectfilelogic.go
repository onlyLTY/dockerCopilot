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
	return &ComposeProjectFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComposeProjectFileLogic) ComposeProjectFile(req *types.ComposeProjectFileReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
