package version

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateprogramLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateprogramLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateprogramLogic {
	return &UpdateprogramLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateprogramLogic) Updateprogram() (resp *types.MsgResp, err error) {
	msg, err := utiles.UpdateProgram(l.svcCtx)
	return &msg, err
}
