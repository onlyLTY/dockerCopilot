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
	return &ComposeProjectFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ComposeProjectFilesLogic) ComposeProjectFiles(req *types.ComposeProjectFileReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
