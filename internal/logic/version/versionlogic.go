package version

import (
	"context"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/config"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type VersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VersionLogic {
	return &VersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VersionLogic) Version() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{
		"version":   config.Version,
		"buildDate": config.BuildDate,
	}
	return resp, nil
}
