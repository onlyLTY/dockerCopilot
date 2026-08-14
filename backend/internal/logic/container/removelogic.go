package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveLogic) Remove(req *types.RemoveContainerReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	taskID := "sync-remove-" + req.Id
	if !l.svcCtx.TryStartContainerDelete(req.Id, taskID) {
		return fail(resp, nil, 409, "该容器正在更新或删除")
	}
	defer l.svcCtx.FinishContainerDelete(req.Id, taskID)
	err = utiles.RemoveContainerWithContext(l.ctx, l.svcCtx, req.Id, req.Force)
	if err != nil {
		l.Errorf("删除容器失败 id=%s force=%v: %v", req.Id, req.Force, err)
		return fail(resp, err, 400, "删除容器失败，容器可能正在运行或已被删除")
	}
	return ok(resp, nil), nil
}
