package container

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

var errContainerUpdateInProgress = errors.New("该容器正在更新，请勿重复提交")

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
	containerID := strings.TrimSpace(req.Id)
	containerName, err := validateContainerName(req.ContainerName)
	if err != nil || containerID == "" {
		resp.Code = 400
		resp.Msg = "容器参数格式错误"
		resp.Data = map[string]interface{}{}
		return resp, errors.New(resp.Msg)
	}
	imageReference, err := imageref.ParseTagged(strings.TrimSpace(req.ImageNameAndTag))
	if err != nil {
		resp.Code = 400
		resp.Msg = "镜像引用格式错误"
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	if !l.svcCtx.BeginContainerUpdate(containerID) {
		resp.Code = 409
		resp.Msg = errContainerUpdateInProgress.Error()
		resp.Data = map[string]interface{}{}
		return resp, errContainerUpdateInProgress
	}
	taskID := uuid.New().String()
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: containerName, Message: "任务已创建",
		DetailMsg: "等待开始更新", Status: svc.TaskStatusRunning,
	})
	go func() {
		defer l.svcCtx.EndContainerUpdate(containerID)
		// Catch any panic and log the error
		defer func() {
			if r := recover(); r != nil {
				l.Errorf("Recovered from panic in UpdateContainer: %v", r)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
					TaskID: taskID, Name: containerName, Message: "更新任务异常终止",
					DetailMsg: fmt.Sprint(r), IsDone: true, Status: svc.TaskStatusFailed,
				})
			}
		}()
		err := utiles.UpdateContainer(l.svcCtx, containerID, containerName, imageReference.Normalized, req.DelOldContainer, taskID)
		if err != nil {
			l.Errorf("Error in UpdateContainer: %v", err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]string{"taskID": taskID}
	return resp, nil
}
