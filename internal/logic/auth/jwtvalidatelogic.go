package auth

import (
	"context"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type JwtValidateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewJwtValidateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JwtValidateLogic {
	return &JwtValidateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JwtValidateLogic) JwtValidate(req *types.LoginReq) (resp *types.Resp, err error) {
	// todo: add your logic here and delete this line

	return
}
