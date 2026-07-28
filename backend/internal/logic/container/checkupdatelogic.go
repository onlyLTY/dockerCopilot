package container

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

type CheckUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUpdateLogic {
	return &CheckUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CheckUpdate 手动触发一次镜像更新检查：异步执行并上报进度，立即返回 taskID。
func (l *CheckUpdateLogic) CheckUpdate() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	taskID := uuid.New().String()
	name := "检查更新"
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: name, Percentage: 0, Message: "任务已提交", DetailMsg: "", IsDone: false,
	})
	svcCtx := l.svcCtx
	go func() {
		defer func() {
			if r := recover(); r != nil {
				svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 0, Message: "检查更新异常", DetailMsg: fmt.Sprintf("%v", r), IsDone: true})
			}
		}()
		list, err := utiles.GetImagesList(svcCtx)
		if err != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 0, Message: "获取镜像列表失败", DetailMsg: err.Error(), IsDone: true})
			return
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 5, Message: "开始检查", DetailMsg: "", IsDone: false})
		svcCtx.HubImageInfo.CheckUpdateWithProgress(list, func(done, total int, current string) {
			pct := 5
			if total > 0 {
				pct = 5 + done*95/total
			}
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Name: name, Percentage: pct,
				Message:   fmt.Sprintf("正在检查 %d/%d", done, total),
				DetailMsg: current, IsDone: false,
			})
		})
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "检查完成", DetailMsg: "", IsDone: true})
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"taskID": taskID}
	return resp, nil
}
