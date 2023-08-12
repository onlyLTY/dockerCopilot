package version

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
