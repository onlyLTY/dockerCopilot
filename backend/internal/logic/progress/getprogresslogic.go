package progress

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

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
	//progress, exists := l.svcCtx.ProgressStore[req.TaskId]
	progress, exists := l.svcCtx.GetProgress(req.TaskId)
	if !exists {
		resp.Code = 400
		resp.Msg = "taskID 未找到"
		resp.Data = map[string]interface{}{}
		return
	}
	resp.Code = 200
	resp.Msg = progress.Message
	resp.Data = map[string]interface{}{
		"taskID":     progress.TaskID,
		"percentage": progress.Percentage,
		"message":    progress.Message,
		"name":       progress.Name,
		"detailMsg":  progress.DetailMsg,
		"isDone":     progress.IsDone,
	}
	return resp, nil
}
