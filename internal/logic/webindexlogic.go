package logic

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type WebindexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWebindexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WebindexLogic {
	return &WebindexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WebindexLogic) Webindex() error {
	// todo: add your logic here and delete this line

	return nil
}
