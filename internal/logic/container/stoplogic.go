package container

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/utiles"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

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
	err = utiles.StopContainer(l.svcCtx, req.Id)
	if err != nil {
		resp.Code = 400
		resp.Msg = err.Error()
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	return resp, nil
}
