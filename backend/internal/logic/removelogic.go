package logic

import (
	"context"
	"strings"

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
	target := strings.TrimSpace(req.Id)
	if !req.Force && strings.TrimSpace(req.RepoTag) != "" {
		target = strings.TrimSpace(req.RepoTag)
	}
	err = utiles.RemoveImage(l.svcCtx, target, req.Force)
	if err != nil {
		l.Errorf("删除镜像失败 target=%s force=%t: %v", target, req.Force, err)
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
