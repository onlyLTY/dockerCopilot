package compose

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
)

const composeProjectBackupTaskName = "批量备份 Compose 项目"

func (l *ComposeProjectBackupLogic) ComposeProjectBackupBatch(req *types.ComposeProjectBackupBatchReq) (*types.Resp, error) {
	resp := &types.Resp{}
	ids := make([]string, 0, len(req.ProjectIDs))
	seen := make(map[string]struct{}, len(req.ProjectIDs))
	for _, id := range req.ProjectIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return errorResp(resp, 400, "请选择要备份的 Compose 项目"), nil
	}

	taskID := uuid.NewString()
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID:  taskID,
		Refresh: true,
		Name:    composeProjectBackupTaskName,
		Message: "任务已提交",
	})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go runComposeProjectBackupBatch(taskCtx, l.svcCtx, taskID, ids)
	return successResp(resp, map[string]interface{}{"taskID": taskID}), nil
}

func runComposeProjectBackupBatch(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID string, projectIDs []string) {
	defer svcCtx.FinishTask(taskID)
	defer func() {
		if recovered := recover(); recovered != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeProjectBackupTaskName, Percentage: 100, Message: "批量备份失败", DetailMsg: fmt.Sprintf("任务异常：%v", recovered), Failed: true, IsDone: true})
		}
	}()
	failed := make([]string, 0)
	for index, projectID := range projectIDs {
		if taskCtx.Err() != nil {
			svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量备份")
			return
		}
		name := "备份项目 " + projectID
		if !svcCtx.TryStartComposeOp(projectID, "backup") {
			detail := "项目正在部署、清理、保存或备份中"
			failed = append(failed, fmt.Sprintf("%s：%s", projectID, detail))
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeProjectBackupTaskName, Percentage: (index + 1) * 100 / len(projectIDs), Message: name, DetailMsg: detail, Failed: true, IsDone: false})
			continue
		}
		func() {
			defer svcCtx.FinishComposeOp(projectID, "backup")
			root, err := composeProject.FindProjectRoot(svcCtx, projectID)
			if err == nil && taskCtx.Err() == nil {
				_, err = composeProject.BackupProject(svcCtx, root)
			}
			percentage := (index + 1) * 100 / len(projectIDs)
			if err != nil {
				if taskCtx.Err() != nil {
					return
				}
				detail := clientMsg(err, "备份失败")
				failed = append(failed, fmt.Sprintf("%s：%s", projectID, detail))
				svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeProjectBackupTaskName, Percentage: percentage, Message: name, DetailMsg: detail, Failed: true, IsDone: false})
				return
			}
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeProjectBackupTaskName, Percentage: percentage, Message: name, DetailMsg: "备份完成", IsDone: false})
		}()
		if taskCtx.Err() != nil {
			svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量备份")
			return
		}
	}
	if taskCtx.Err() != nil {
		svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量备份")
		return
	}
	detail := "全部项目备份完成"
	if len(failed) > 0 {
		detail = "失败项目：" + strings.Join(failed, "；")
	}
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: composeProjectBackupTaskName, Percentage: 100, Message: "批量备份完成", DetailMsg: detail, Failed: len(failed) > 0, IsDone: true})
}
