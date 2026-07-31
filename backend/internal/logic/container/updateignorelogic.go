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
		errMsg := "无法获取容器名称"
		if err != nil {
			errMsg = err.Error()
		}
		resp.Code = 404
		resp.Msg = errMsg
		return resp, errorx.NewCodeError(404, errMsg)
	}
	name := inspected.Name[1:]
	if err = settingstore.SetContainerUpdateIgnored(name, ignored); err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		return resp, errorx.NewCodeError(500, err.Error())
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{"name": name, "ignored": ignored}
	return resp, nil
}
