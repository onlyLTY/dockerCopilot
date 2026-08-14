package progress

import (
	"context"
	"sort"

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
	stored := l.svcCtx.ListProgress()
	progresses := make([]svc.TaskProgress, 0, len(stored))
	for _, progress := range stored {
		progresses = append(progresses, progress)
	}
	sort.Slice(progresses, func(i, j int) bool {
		if progresses[i].UpdatedAt == progresses[j].UpdatedAt {
			return progresses[i].TaskID > progresses[j].TaskID
		}
		return progresses[i].UpdatedAt > progresses[j].UpdatedAt
	})
	items := make([]map[string]interface{}, 0, len(progresses))
	for _, progress := range progresses {
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
		"progressType":   progress.ProgressType,
		"indeterminate":  progress.Indeterminate,
		"current":        progress.Current,
		"total":          progress.Total,

		"steps":     progress.Steps,
		"isDone":    progress.IsDone,
		"failed":    progress.Failed,
		"canceled":  progress.Canceled,
		"timedOut":  progress.TimedOut,
		"updatedAt": progress.UpdatedAt,
	}
}
