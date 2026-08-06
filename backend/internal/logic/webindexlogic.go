package logic

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
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
	// 实际跳转在 handler；logic 仅满足 goctl 分层占位。
	return nil
}
