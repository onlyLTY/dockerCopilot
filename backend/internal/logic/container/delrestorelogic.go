package container

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type DelRestoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDelRestoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DelRestoreLogic {
	return &DelRestoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DelRestoreLogic) DelRestore(req *types.DelContainerBackupReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	fileName, err := url.QueryUnescape(strings.TrimSpace(req.Filename))
	if err != nil {
		resp.Code = 400
		resp.Msg = "文件名解码失败"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	if _, err := utiles.ResolveBackupPath(fileName); err != nil {
		l.Errorf("删除备份文件失败，filename=%q, error=%v", fileName, err)
		resp.Code = 400
		resp.Msg = "备份文件名不合法"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	taskID := uuid.NewString()
	name := "删除容器备份"
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Refresh: true, Name: name, Message: "任务已提交"})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go runDeleteBackupTask(taskCtx, l.svcCtx, taskID, name, fileName)
	return ok(resp, map[string]interface{}{"taskID": taskID}), nil
}

func CleanFilename(filename string) string {
	return filename
}
