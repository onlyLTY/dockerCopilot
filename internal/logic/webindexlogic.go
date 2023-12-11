package logic

import (
	"context"

	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
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
	// 这个logic没啥用 别看了 要看就去看handler
	return nil
}
