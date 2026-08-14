package container

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
)

const batchRemoveContainersTaskName = "批量删除容器"

func (l *RemoveLogic) RemoveBatch(req *types.BatchRemoveContainersReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if !req.Confirm {
		return fail(resp, nil, 400, "必须明确确认删除操作")
	}
	ids := make([]string, 0, len(req.ContainerIDs))
	seen := make(map[string]struct{}, len(req.ContainerIDs))
	for _, id := range req.ContainerIDs {
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
		return fail(resp, nil, 400, "请选择要删除的容器")
	}
	if err := l.svcCtx.RequireDocker(); err != nil {
		return fail(resp, err, 503, "Docker 服务不可用")
	}
	taskID := uuid.NewString()
	for _, id := range ids {
		if !l.svcCtx.TryStartContainerDelete(id, taskID) {
			for _, reserved := range ids {
				if reserved == id {
					break
				}
				l.svcCtx.FinishContainerDelete(reserved, taskID)
			}
			return fail(resp, nil, 409, "所选容器中有容器正在删除")
		}
	}
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: batchRemoveContainersTaskName, Message: "任务已提交"})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go runBatchRemoveContainers(taskCtx, l.svcCtx, taskID, ids)
	return ok(resp, map[string]interface{}{"taskID": taskID}), nil
}

func runBatchRemoveContainers(taskCtx context.Context, svcCtx *svc.ServiceContext, taskID string, ids []string) {
	defer svcCtx.FinishTask(taskID)
	defer func() {
		for _, id := range ids {
			svcCtx.FinishContainerDelete(id, taskID)
		}
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: batchRemoveContainersTaskName, Percentage: 100, Message: "批量删除失败", DetailMsg: "任务异常", Failed: true, IsDone: true})
		}
	}()
	failed := make([]string, 0)
	for index, id := range ids {
		if taskCtx.Err() != nil {
			svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除容器")
			return
		}
		name := "删除容器 " + id
		inspect, err := svcCtx.DockerClient.ContainerInspect(taskCtx, id)
		force := err == nil && inspect.State != nil && inspect.State.Running
		if err == nil {
			err = utiles.RemoveContainerWithContext(taskCtx, svcCtx, id, force)
		}
		percentage := (index + 1) * 100 / len(ids)
		if err != nil {
			if taskCtx.Err() != nil {
				svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除容器")
				return
			}
			failed = append(failed, id+"："+err.Error())
			svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: batchRemoveContainersTaskName, Percentage: percentage, Message: name, DetailMsg: "删除失败：" + err.Error(), Failed: true})
			continue
		}
		svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: batchRemoveContainersTaskName, Percentage: percentage, Message: name, DetailMsg: "删除完成"})
	}
	if taskCtx.Err() != nil {
		svcCtx.MarkTaskCanceled(taskID, "用户请求停止批量删除容器")
		return
	}
	detail := "全部容器删除完成"
	if len(failed) > 0 {
		detail = "失败容器：" + strings.Join(failed, "；")
	}
	svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: batchRemoveContainersTaskName, Percentage: 100, Message: "批量删除完成", DetailMsg: detail, Failed: len(failed) > 0, IsDone: true})
}
