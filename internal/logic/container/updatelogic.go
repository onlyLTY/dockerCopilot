package container

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

var errContainerUpdateInProgress = errors.New("该容器正在更新，请勿重复提交")
var errSelfContainerUpdate = errors.New("当前 Docker Copilot 不能在自身容器内执行原地更新，请拉取新镜像后由 Docker Compose 重新创建")

const containerUpdateQueueTimeout = 30 * time.Minute

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
	if utiles.IsSelfContainerID(containerID) {
		resp.Code = 409
		resp.Msg = errSelfContainerUpdate.Error()
		resp.Data = map[string]interface{}{}
		return resp, errSelfContainerUpdate
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
		l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
			TaskID: taskID, Name: containerName, Message: "等待更新执行槽",
			DetailMsg: "最多同时更新两个容器", Status: svc.TaskStatusRunning,
		})
		queueContext, cancelQueue := context.WithTimeout(context.Background(), containerUpdateQueueTimeout)
		acquired := l.svcCtx.AcquireContainerUpdateSlot(queueContext)
		cancelQueue()
		if !acquired {
			l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID: taskID, Name: containerName, Message: "更新任务已取消",
				DetailMsg: "等待执行槽时任务被取消", IsDone: true, Status: svc.TaskStatusFailed,
			})
			return
		}
		defer l.svcCtx.ReleaseContainerUpdateSlot()
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
