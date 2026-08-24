package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopLogic {
	return &StopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StopLogic) Stop(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	containerID, err := beginContainerOperation(l.svcCtx, resp, req.Id, "stop")
	if err != nil {
		return resp, err
	}
	defer l.svcCtx.EndContainerOperation(containerID)
	err = utiles.StopContainer(l.svcCtx, containerID)
	if err != nil {
		resp.Code = 400
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
