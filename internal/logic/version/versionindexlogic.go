package version

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type VersionIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVersionIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VersionIndexLogic {
	return &VersionIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VersionIndexLogic) VersionIndex() error {
	// todo: add your logic here and delete this line

	return nil
}
