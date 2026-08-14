package container

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreLogic {
	return &RestoreLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RestoreLogic) Restore(req *types.ContainerRestoreReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	taskID := uuid.New().String()
	name := "恢复容器"
	fileName := req.Filename
	if _, err := utiles.ResolveBackupPath(fileName); err != nil {
		resp.Code = 400
		resp.Msg = "备份文件名不合法"
		resp.Data = map[string]interface{}{}
		return resp, errorx.NewCodeError(400, "备份文件名不合法")
	}
	if strings.ToLower(filepath.Ext(fileName)) != ".json" {
		err = errorx.NewCodeError(400, "目前仅支持config备份恢复")
		resp.Code = 400
		resp.Msg = "目前仅支持config备份恢复"
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Refresh: true, Name: name, Message: "任务已提交", DetailMsg: "", IsDone: false})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go func() {
		defer l.svcCtx.FinishTask(taskID)

		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("恢复容器异常: %v", r)
				l.Errorf("task=%s file=%s %s", taskID, fileName, message)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Message: "恢复失败", DetailMsg: message, IsDone: true})
			}
		}()
		if err := utiles.RestoreContainerWithContext(taskCtx, l.svcCtx, fileName, taskID); err != nil {
			if taskCtx.Err() != nil {
				l.svcCtx.MarkTaskCanceled(taskID, "任务已停止；已恢复的容器不会自动回滚")
				return
			}
			l.Errorf("restore container failed task=%s file=%s: %v", taskID, fileName, err)
			l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
				TaskID:     taskID,
				Name:       name,
				Percentage: 100,
				Message:    "恢复失败",
				DetailMsg:  err.Error(),
				Failed:     true,
				IsDone:     true,
			})
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{"taskID": taskID}
	return resp, nil
}
