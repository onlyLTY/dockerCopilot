package progress

import (
	"context"

	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProgressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProgressLogic {
	return &GetProgressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProgressLogic) GetProgress(req *types.GetProgressReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	progress, exists := l.svcCtx.ProgressStore[req.TaskId]
	if !exists {
		resp.Code = 400
		resp.Msg = "taskID 未找到"
		return
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"schedule": progress.Message}
	if progress.IsDone {
		delete(l.svcCtx.ProgressStore, req.TaskId)
	}
	return resp, nil
}
