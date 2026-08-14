package container

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func (l *BackupLogic) submitBackupTask(name string, work func(context.Context) error) (*types.Resp, error) {
	return submitBackupTask(l.svcCtx, name, work)
}

func (l *Backup2composeLogic) submitBackupTask(name string, work func(context.Context) error) (*types.Resp, error) {
	return submitBackupTask(l.svcCtx, name, work)
}

func submitBackupTask(svcCtx *svc.ServiceContext, name string, work func(context.Context) error) (*types.Resp, error) {
	resp := &types.Resp{}
	taskID := uuid.NewString()
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Refresh: true, Name: name, Message: "任务已提交"})
	taskCtx := svcCtx.RegisterTask(taskID)
	go func() {
		defer svcCtx.FinishTask(taskID)
		defer func() {
			if recovered := recover(); recovered != nil {
				svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "任务失败", DetailMsg: fmt.Sprintf("备份任务异常：%v", recovered), Failed: true, IsDone: true})
			}
		}()
		if err := work(taskCtx); err != nil {
			if taskCtx.Err() != nil {
				svcCtx.MarkTaskCanceled(taskID, "任务已停止；已生成的备份文件不会自动回滚")
				return
			}
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Name: name, Percentage: 100,
				Message: "任务失败", DetailMsg: backupTaskError(err), Failed: true, IsDone: true,
			})
			return
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{
			TaskID: taskID, Name: name, Percentage: 100,
			Message: "任务完成", DetailMsg: "备份已完成", IsDone: true,
		})
	}()
	resp.Code, resp.Msg, resp.Data = 200, "success", map[string]interface{}{"taskID": taskID}
	return resp, nil
}

func backupTaskError(err error) string {
	if err == nil {
		return "备份失败"
	}
	return fmt.Sprintf("备份失败：%v", err)
}
