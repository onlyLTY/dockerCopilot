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
	progress, exists := l.svcCtx.GetProgress(req.TaskId)
	if !exists {
		resp.Code = 400
		resp.Msg = "taskID 未找到"
		resp.Data = map[string]interface{}{}
		return
	}
	resp.Code = 200
	resp.Msg = progress.Message
	resp.Data = progressData(progress)
	return resp, nil
}

func (l *GetProgressLogic) ListProgress() (resp *types.Resp, err error) {
	items := make([]map[string]interface{}, 0)
	for _, progress := range l.svcCtx.ListProgress() {
		items = append(items, progressData(progress))
	}
	resp = &types.Resp{Code: 200, Msg: "success", Data: items}
	return resp, nil
}

func progressData(progress svc.TaskProgress) map[string]interface{} {
	return map[string]interface{}{
		"taskID":         progress.TaskID,
		"resourceID":     progress.ResourceID,
		"percentage":     progress.Percentage,
		"message":        progress.Message,
		"name":           progress.Name,
		"detailMsg":      progress.DetailMsg,
		"stepPercentage": progress.StepPercentage,

		"steps":     progress.Steps,
		"isDone":    progress.IsDone,
		"updatedAt": progress.UpdatedAt,
	}
}
