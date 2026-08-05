package logic

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveLogic {
	return &RemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RemoveLogic) Remove(req *types.RemoveImageReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.RemoveImage(l.svcCtx, req.Id, req.Force)
	if err != nil {
		l.Errorf("删除镜像失败 id=%s: %v", req.Id, err)
		if errorx.IsDockerUnavailable(err) {
			resp.Code = errorx.CodeDockerUnavailable
			resp.Msg = "Docker 服务不可用"
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 409
		resp.Msg = "删除镜像失败，镜像可能正在使用或已被删除"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
