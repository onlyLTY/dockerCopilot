package progress

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProgressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProgressLogic {
	return &DeleteProgressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProgressLogic) DeleteProgress(req *types.DeleteProgressReq) (resp *types.Resp, err error) {
	l.svcCtx.DeleteProgress(req.TaskId)
	return &types.Resp{Code: 200, Msg: "success"}, nil
}

func (l *DeleteProgressLogic) CancelProgress(req *types.CancelProgressReq) (resp *types.Resp, err error) {
	progress, exists, active := l.svcCtx.CancelTask(req.TaskId)
	if !exists {
		return &types.Resp{Code: 404, Msg: "任务不存在", Data: map[string]interface{}{}}, nil
	}
	if progress.IsDone {
		return &types.Resp{Code: 409, Msg: "任务已结束", Data: progress}, nil
	}
	if !active {
		return &types.Resp{Code: 409, Msg: "任务当前不可停止", Data: progress}, nil
	}
	return &types.Resp{Code: 202, Msg: "已请求停止任务", Data: map[string]string{"taskID": req.TaskId}}, nil
}
func (l *DeleteProgressLogic) ClearProgress(req *types.ClearProgressReq) (resp *types.Resp, err error) {
	l.svcCtx.ClearProgress(req.DoneOnly)
	return &types.Resp{Code: 200, Msg: "success"}, nil
}
