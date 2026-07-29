package compose

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	composeProject "github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
	"github.com/zeromicro/go-zero/core/logx"
)

type ProjectsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProjectsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectsListLogic {
	return &ProjectsListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ProjectsListLogic) List() (*types.Resp, error) {
	resp := &types.Resp{}
	if l.svcCtx.DockerClient == nil {
		logx.Errorf("compose operation=project_list failed=docker service unavailable")
		resp.Code = 503
		resp.Msg = "Docker 服务不可用"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	data, err := composeProject.ScanProjects(l.ctx, l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = "读取 Compose 项目失败"
		resp.Data = map[string]interface{}{}
		logx.Errorf("compose operation=project_list failed=scan error=%v", err)
		return resp, err
	}
	logx.Infof("compose operation=project_list success")
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = data
	return resp, nil
}

type PortsListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPortsListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PortsListLogic {
	return &PortsListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *PortsListLogic) List() (*types.Resp, error) {
	resp := &types.Resp{}
	if l.svcCtx.DockerClient == nil {
		logx.Errorf("compose operation=ports_list failed=docker service unavailable")
		resp.Code = 503
		resp.Msg = "Docker 服务不可用"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	data, err := composeProject.ListPorts(l.ctx, l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = "读取端口使用情况失败"
		resp.Data = map[string]interface{}{}
		logx.Errorf("compose operation=ports_list failed=list error=%v", err)
		return resp, err
	}
	logx.Infof("compose operation=ports_list success")
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = data
	return resp, nil
}
