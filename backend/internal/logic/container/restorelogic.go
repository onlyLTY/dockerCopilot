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
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Message: "任务已提交", DetailMsg: "", IsDone: false})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("恢复容器异常: %v", r)
				l.Errorf("task=%s file=%s %s", taskID, fileName, message)
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Message: "恢复失败", DetailMsg: message, IsDone: true})
			}
		}()
		if err := utiles.RestoreContainer(l.svcCtx, fileName, taskID); err != nil {
			l.Errorf("restore container failed task=%s file=%s: %v", taskID, fileName, err)
		}
	}()
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{"taskID": taskID}
	return resp, nil
}
