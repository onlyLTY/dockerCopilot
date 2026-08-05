package container

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateIgnoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateIgnoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateIgnoreLogic {
	return &UpdateIgnoreLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateIgnoreLogic) Set(req *types.ContainerUpdateIgnoreReq, ignored bool) (*types.Resp, error) {
	resp := &types.Resp{}
	if err := l.svcCtx.RequireDocker(); err != nil {
		return fail(resp, err, errorx.CodeDockerUnavailable, "Docker 服务不可用")
	}
	inspected, err := l.svcCtx.DockerClient.ContainerInspect(l.ctx, req.Id)
	if err != nil || inspected.Name == "" {
		if err != nil {
			l.Errorf("获取容器名称失败 id=%s: %v", req.Id, err)
		}
		return fail(resp, errorx.NewCodeError(404, "无法获取容器名称"), 404, "无法获取容器名称")
	}
	name := inspected.Name[1:]
	if err = settingstore.SetContainerUpdateIgnored(name, ignored); err != nil {
		l.Errorf("更新容器忽略状态失败 name=%s: %v", name, err)
		return fail(resp, errorx.NewCodeError(500, "更新忽略状态失败"), 500, "更新忽略状态失败")
	}
	return ok(resp, map[string]interface{}{"name": name, "ignored": ignored}), nil
}
