package container

import (
	"context"
	"net/url"
	"os"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type DelRestoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDelRestoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DelRestoreLogic {
	return &DelRestoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DelRestoreLogic) DelRestore(req *types.DelContainerBackupReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	fileName, err := url.QueryUnescape(req.Filename)
	if err != nil {
		resp.Code = 400
		resp.Msg = "文件名解码失败"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	fullPath, err := utiles.ResolveBackupPath(fileName)
	if err != nil {
		resp.Code = 400
		resp.Msg = "备份文件名不合法"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	err = os.Remove(fullPath)
	if err != nil {
		resp.Code = 400
		resp.Msg = "删除失败"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}

func CleanFilename(filename string) string {
	return filename
}
