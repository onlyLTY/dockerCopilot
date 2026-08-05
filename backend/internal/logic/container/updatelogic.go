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
	taskID := uuid.New().String()
	name := req.ContainerName
	if name == "" {
		name = req.Id
	}
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: "更新 " + name, Message: "任务已提交", DetailMsg: "", IsDone: false})
	if err := l.svcCtx.RequireDocker(); err != nil {
		return fail(resp, err, 503, "Docker 服务不可用")
	}
	if !l.svcCtx.TryStartContainerUpdate(req.Id, taskID) {
		resp.Code = 409
		resp.Msg = "该容器正在更新"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	go func() {
		defer l.svcCtx.FinishContainerUpdate(req.Id, taskID)
		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("更新容器异常: %v", r)
				l.Errorf("task=%s container=%s %s", taskID, req.Id, message)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: "更新 " + name, Message: "更新失败", DetailMsg: message, IsDone: true})
			}
		}()
		imageNameAndTag := req.ImageNameAndTag
		if imageNameAndTag == "" {
			inspected, inspectErr := utiles.GetContainerInspect(l.svcCtx, req.Id)
			if inspectErr != nil || inspected.Config == nil || inspected.Config.Image == "" {
				message := "无法从容器获取镜像名称"
				if inspectErr != nil {
					message = inspectErr.Error()
				}
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: "更新 " + name, Message: "更新失败", DetailMsg: message, IsDone: true})
				return
			}
			imageNameAndTag = inspected.Config.Image
		}
		err := utiles.UpdateContainer(l.svcCtx, req.Id, name, imageNameAndTag, os.Getenv("DelOldContainer") != "false", taskID)
		if err != nil {
			l.Errorf("update container failed task=%s container=%s: %v", taskID, req.Id, err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"taskID": taskID}
	return resp, nil
}
