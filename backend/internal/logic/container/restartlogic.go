package container

import (
	"context"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestartLogic {
	return &RestartLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RestartLogic) Restart(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if !l.svcCtx.TryStartContainerUpdate(req.Id, "direct-restart") {
		return fail(resp, nil, 409, "容器当前正被其他操作占用")
	}
	defer l.svcCtx.FinishContainerUpdate(req.Id, "direct-restart")
	operationCtx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()
	err = utiles.RestartContainerWithContext(operationCtx, l.svcCtx, req.Id)
	if err != nil {
		l.Errorf("重启容器失败 id=%s: %v", req.Id, err)
		return fail(resp, err, 400, "重启容器失败")
	}
	return ok(resp, nil), nil
}
