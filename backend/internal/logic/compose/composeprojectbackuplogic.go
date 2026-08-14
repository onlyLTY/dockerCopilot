package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectBackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectBackupLogic {
	return &ComposeProjectBackupLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectBackupLogic) ComposeProjectBackup(req *types.ComposeProjectIdReq) (*types.Resp, error) {
	resp := &types.Resp{}
	root, err := composeProject.FindProjectRoot(l.svcCtx, req.ProjectID)
	if err != nil {
		return errorResp(resp, 404, clientMsg(err, "Compose 项目不存在")), nil
	}
	if !l.svcCtx.TryStartComposeOp(req.ProjectID, "backup") {
		return errorResp(resp, 409, "该项目正在部署、清理、保存或备份中，请稍后再试"), nil
	}
	defer l.svcCtx.FinishComposeOp(req.ProjectID, "backup")
	backup, err := composeProject.BackupProject(l.svcCtx, root)
	if err != nil {
		logx.Errorf("compose operation=backup project=%s failed=%v", req.ProjectID, err)
		return errorResp(resp, 500, clientMsg(err, "备份 Compose 项目失败")), nil
	}
	logx.Infof("compose operation=backup project=%s success filename=%v", req.ProjectID, backup["filename"])
	return successResp(resp, backup), nil
}
