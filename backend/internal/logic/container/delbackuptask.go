package container

import (
	"context"
	"os"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
)

func runDeleteBackupTask(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID, name, filename string) {
	defer svcCtx.FinishTask(taskID)
	defer func() {
		if recovered := recover(); recovered != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "任务失败", DetailMsg: "删除备份任务异常", Failed: true, IsDone: true})
		}
	}()
	if err := taskCtx.Err(); err != nil {
		svcCtx.MarkTaskCanceled(taskID, "任务已停止；备份文件未删除")
		return
	}
	path, err := utiles.ResolveBackupPath(filename)
	if err == nil {
		err = os.Remove(path)
	}
	if err != nil {
		if taskCtx.Err() != nil {
			svcCtx.MarkTaskCanceled(taskID, "任务已停止；备份文件未删除")
			return
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "任务失败", DetailMsg: "删除备份失败：" + err.Error(), Failed: true, IsDone: true})
		return
	}
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "任务完成", DetailMsg: "备份已删除", IsDone: true})
}
