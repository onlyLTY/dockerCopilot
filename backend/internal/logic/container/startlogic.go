package container

import (
	"context"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartLogic {
	return &StartLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *StartLogic) Start(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if !l.svcCtx.TryStartContainerUpdate(req.Id, "direct-start") {
		return fail(resp, nil, 409, "容器当前正被其他操作占用")
	}
	defer l.svcCtx.FinishContainerUpdate(req.Id, "direct-start")
	operationCtx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()
	err = utiles.StartContainerWithContext(operationCtx, l.svcCtx, req.Id)
	if err != nil {
		l.Errorf("启动容器失败 id=%s: %v", req.Id, err)
		return fail(resp, err, 400, "启动容器失败")
	}
	return ok(resp, nil), nil
}
