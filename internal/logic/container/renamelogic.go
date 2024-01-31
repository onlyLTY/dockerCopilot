package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RenameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRenameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RenameLogic {
	return &RenameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RenameLogic) Rename(req *types.ContainerRenameReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.RenameContainer(l.svcCtx, req.Id, req.NewName)
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
