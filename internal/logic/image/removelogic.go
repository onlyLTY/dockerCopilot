package image

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/utiles"
	"strings"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

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
	var imageId = req.Id
	if strings.HasPrefix(imageId, "sha256:") {
		imageId = strings.TrimPrefix(imageId, "sha256:")
	}
	err = utiles.RemoveImage(l.svcCtx, req.Id, req.Force)
	if err != nil {
		resp.Code = 409
		resp.Msg = err.Error()
		return resp, nil
	}
	resp.Code = 200
	resp.Msg = "success"
	return resp, nil
}
