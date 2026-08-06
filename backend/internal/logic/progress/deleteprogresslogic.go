package progress

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProgressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProgressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProgressLogic {
	return &DeleteProgressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProgressLogic) DeleteProgress(req *types.DeleteProgressReq) (resp *types.Resp, err error) {
	l.svcCtx.DeleteProgress(req.TaskId)
	return &types.Resp{Code: 200, Msg: "success"}, nil
}

func (l *DeleteProgressLogic) ClearProgress(req *types.ClearProgressReq) (resp *types.Resp, err error) {
	l.svcCtx.ClearProgress(req.DoneOnly)
	return &types.Resp{Code: 200, Msg: "success"}, nil
}