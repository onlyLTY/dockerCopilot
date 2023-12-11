package container

import (
	"context"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartLogic {
	return &StartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartLogic) Start(req *types.IdReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.StartContainer(l.svcCtx, req.Id)
	if err != nil {
		resp.Code = 400
		resp.Msg = err.Error()
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	return resp, nil
}
