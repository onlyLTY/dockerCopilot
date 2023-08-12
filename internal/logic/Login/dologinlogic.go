package Login

import (
	"context"
	"errors"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DoLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDoLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DoLoginLogic {
	return &DoLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DoLoginLogic) DoLogin(req *types.DoLoginReq) error {
	// todo: add your logic here and delete this line
	if req.SecretKey != l.svcCtx.Config.SecretKey {
		return errors.New("秘钥错误")
	}
	return nil
}
