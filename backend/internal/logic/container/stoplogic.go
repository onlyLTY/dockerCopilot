package container

import (
	"context"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopLogic {
	return &StopLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *StopLogic) Stop(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if !l.svcCtx.TryStartContainerUpdate(req.Id, "direct-stop") {
		return fail(resp, nil, 409, "容器当前正被其他操作占用")
	}
	defer l.svcCtx.FinishContainerUpdate(req.Id, "direct-stop")
	operationCtx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()
	err = utiles.StopContainerWithContext(operationCtx, l.svcCtx, req.Id)
	if err != nil {
		l.Errorf("停止容器失败 id=%s: %v", req.Id, err)
		return fail(resp, err, 400, "停止容器失败")
	}
	return ok(resp, nil), nil
}
