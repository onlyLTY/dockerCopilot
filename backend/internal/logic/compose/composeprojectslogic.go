package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	"github.com/zeromicro/go-zero/core/logx"
)

type ComposeProjectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewComposeProjectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ComposeProjectsLogic {
	return &ComposeProjectsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ComposeProjectsLogic) ComposeProjects() (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		logx.Errorf("compose operation=project_list failed=docker service unavailable")
		return errorResp(resp, errorx.CodeDockerUnavailable, "Docker 服务不可用"), err
	}
	data, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
		if errorx.IsDockerUnavailable(err) {
			return errorResp(resp, errorx.CodeDockerUnavailable, "Docker 服务不可用"), err
		}
		logx.Errorf("compose operation=project_list failed=scan error=%v", err)
		return errorResp(resp, 500, "读取 Compose 项目失败"), err
	}
	logx.Infof("compose operation=project_list success")
	return successResp(resp, data), nil
}
