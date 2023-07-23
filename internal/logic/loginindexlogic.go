package logic

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginIndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginIndexLogic {
	return &LoginIndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginIndexLogic) LoginIndex() error {
	// todo: add your logic here and delete this line

	return nil
}
