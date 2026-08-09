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
	go func() {
		defer l.svcCtx.FinishContainerUpdate(containerID, taskID)
		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("更新容器异常: %v", r)
				l.Errorf("task=%s container=%s %s", taskID, containerID, message)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, ResourceID: containerID, Name: "更新 " + name, Message: "更新失败", DetailMsg: message, IsDone: true})
			}
		}()

		l.svcCtx.AcquireContainerUpdateSlot()
		defer l.svcCtx.ReleaseContainerUpdateSlot()
		queued, _ := l.svcCtx.GetProgress(taskID)
		queued.Message = "开始执行更新"
		queued.DetailMsg = "已获得 Docker 执行槽位"
		l.svcCtx.UpdateProgress(taskID, queued)

		currentImage := imageNameAndTag
		if currentImage == "" {
			inspected, inspectErr := utiles.GetContainerInspect(l.svcCtx, containerID)
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
		if err := utiles.UpdateContainer(l.svcCtx, containerID, name, currentImage, delOldContainer, taskID); err != nil {
			l.Errorf("update container failed task=%s container=%s: %v", taskID, containerID, err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"taskID": taskID}
	return resp, nil
}
