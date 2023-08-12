package version

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/config"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetVersionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVersionsLogic {
	return &GetVersionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVersionsLogic) GetVersions() (resp *types.VersionMsgResp, err error) {
	return &types.VersionMsgResp{
		Version:   config.Version,
		BuildDate: config.BuildDate,
	}, nil
}
