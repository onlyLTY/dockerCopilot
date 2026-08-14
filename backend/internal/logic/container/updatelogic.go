package container

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
)

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.ContainerUpdateReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		return fail(resp, err, 503, "Docker 服务不可用")
	}

	taskID := uuid.New().String()
	containerID := req.Id
	name := req.ContainerName
	if name == "" {
		name = containerID
	}
	if !l.svcCtx.TryStartContainerUpdate(containerID, taskID) {
		resp.Code = 409
		resp.Msg = "该容器正在更新"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}

	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID:     taskID,
		ResourceID: containerID,
		Name:       "更新 " + name,
		Message:    "任务已提交",
		IsDone:     false,
	})
	imageNameAndTag := req.ImageNameAndTag
	delOldContainer := os.Getenv("DelOldContainer") != "false"
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go func() {
		defer l.svcCtx.FinishTask(taskID)
		defer l.svcCtx.FinishContainerUpdate(containerID, taskID)
		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("更新容器异常: %v", r)
				l.Errorf("task=%s container=%s %s", taskID, containerID, message)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, ResourceID: containerID, Name: "更新 " + name, Message: "更新失败", DetailMsg: message, IsDone: true})
			}
		}()

		if !l.svcCtx.AcquireContainerUpdateSlot(taskCtx) {
			l.svcCtx.MarkTaskCanceled(taskID, "任务已停止，尚未开始执行 Docker 操作")
			return
		}
		defer l.svcCtx.ReleaseContainerUpdateSlot()
		if err := taskCtx.Err(); err != nil {
			l.svcCtx.MarkTaskCanceled(taskID, "任务已停止，尚未开始执行 Docker 操作")
			return
		}
		queued, _ := l.svcCtx.GetProgress(taskID)
		queued.Message = "开始执行更新"
		queued.DetailMsg = "已获得 Docker 执行槽位"
		l.svcCtx.UpdateProgress(taskID, queued)

		currentImage := imageNameAndTag
		if currentImage == "" {
			inspected, inspectErr := utiles.GetContainerInspectWithContext(taskCtx, l.svcCtx, containerID)
			if inspectErr != nil || inspected.Config == nil || inspected.Config.Image == "" {
				message := "无法从容器获取镜像名称"
				if inspectErr != nil {
					message = inspectErr.Error()
				}
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, ResourceID: containerID, Name: "更新 " + name, Message: "更新失败", DetailMsg: message, IsDone: true})
				return
			}
			currentImage = inspected.Config.Image
		}
		if err := utiles.UpdateContainerWithContext(taskCtx, l.svcCtx, containerID, name, currentImage, delOldContainer, taskID); err != nil {
			if taskCtx.Err() != nil {
				l.svcCtx.MarkTaskCanceled(taskID, "任务已停止；已执行的 Docker 操作不会自动回滚")
				return
			}
			l.Errorf("update container failed task=%s container=%s: %v", taskID, containerID, err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"taskID": taskID}
	return resp, nil
}
