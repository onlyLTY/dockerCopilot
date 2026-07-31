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
	return &ComposeDeployLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComposeDeployLogic) ComposeDeploy(req *types.ComposeDeployReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
