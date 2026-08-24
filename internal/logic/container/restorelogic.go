package container

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var errRestoreInProgress = errors.New("已有恢复任务正在执行")

type RestoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreLogic {
	return &RestoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestoreLogic) Restore(req *types.ContainerRestoreReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	taskID := uuid.New().String()
	fileName := req.Filename
	fullPath, pathErr := utiles.ResolveBackupPath(fileName, ".json")
	if pathErr != nil {
		resp.Code = 400
		resp.Msg = pathErr.Error()
		resp.Data = map[string]interface{}{}
		return resp, pathErr
	}
	fileName = filepath.Base(fullPath)
	if filepath.Ext(fileName) != ".json" {
		err = fmt.Errorf("目前仅支持config备份恢复")
		resp.Code = 400
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	if !l.svcCtx.BeginRestore() {
		resp.Code = 409
		resp.Msg = errRestoreInProgress.Error()
		resp.Data = map[string]interface{}{}
		return resp, errRestoreInProgress
	}
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: "恢复容器", Message: "任务已创建",
		DetailMsg: "等待开始恢复", Status: svc.TaskStatusRunning,
	})
	go func() {
		defer l.svcCtx.EndRestore()
		// Catch any panic and log the error
		defer func() {
			if r := recover(); r != nil {
				l.Errorf("Recovered from panic in restoreContainer: %v", r)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{
					TaskID: taskID, Name: "恢复容器", Message: "恢复任务异常终止",
					DetailMsg: fmt.Sprint(r), IsDone: true, Status: svc.TaskStatusFailed,
				})
			}
		}()
		err := utiles.RestoreContainer(l.svcCtx, fileName, taskID)
		if err != nil {
			l.Errorf("Error in restoreContainer: %v", err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"taskID": taskID,
	}
	return resp, nil
}
