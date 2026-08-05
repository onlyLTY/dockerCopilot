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
	resp := &types.Resp{Data: map[string]interface{}{}}
	if l.svcCtx.DockerClient == nil {
		resp.Code = 500
		resp.Msg = "Docker 客户端不可用"
		return resp, errorx.NewCodeError(500, "Docker 客户端不可用")
	}
	inspected, err := l.svcCtx.DockerClient.ContainerInspect(l.ctx, req.Id)
	if err != nil || inspected.Name == "" {
		if err != nil {
			l.Errorf("获取容器名称失败 id=%s: %v", req.Id, err)
		}
		resp.Code = 404
		resp.Msg = "无法获取容器名称"
		return resp, errorx.NewCodeError(404, "无法获取容器名称")
	}
	name := inspected.Name[1:]
	if err = settingstore.SetContainerUpdateIgnored(name, ignored); err != nil {
		l.Errorf("更新容器忽略状态失败 name=%s: %v", name, err)
		resp.Code = 500
		resp.Msg = "更新忽略状态失败"
		return resp, errorx.NewCodeError(500, "更新忽略状态失败")
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{"name": name, "ignored": ignored}
	return resp, nil
}
