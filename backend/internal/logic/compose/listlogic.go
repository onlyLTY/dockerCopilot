package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	"github.com/zeromicro/go-zero/core/logx"
)

// ComposeProjectsLogic 项目列表（goctl 命名；实现与旧 ProjectsListLogic 合并为一套）。
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

// PortsLogic 端口汇总。
type PortsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPortsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PortsLogic {
	return &PortsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *PortsLogic) Ports() (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		logx.Errorf("compose operation=ports_list failed=docker service unavailable")
		return errorResp(resp, errorx.CodeDockerUnavailable, "Docker 服务不可用"), err
	}
	data, err := composeProject.ListPorts(l.ctx, l.svcCtx)
	if err != nil {
		if errorx.IsDockerUnavailable(err) {
			return errorResp(resp, errorx.CodeDockerUnavailable, "Docker 服务不可用"), err
		}
		logx.Errorf("compose operation=ports_list failed=list error=%v", err)
		return errorResp(resp, 500, "读取端口使用情况失败"), err
	}
	logx.Infof("compose operation=ports_list success")
	return successResp(resp, data), nil
}
