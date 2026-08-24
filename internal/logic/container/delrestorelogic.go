package container

import (
	"context"
	"os"
	"path/filepath"

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
	fullPath, err := utiles.ResolveBackupPath(req.Filename, ".json", ".yaml")
	if err != nil {
		resp.Code = 400
		resp.Msg = "非法文件名"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	backupRoot, openErr := os.OpenRoot(filepath.Dir(fullPath))
	if openErr != nil {
		resp.Code = 500
		resp.Msg = "删除失败"
		resp.Data = map[string]interface{}{}
		return resp, nil
	}
	defer backupRoot.Close()
	err = backupRoot.Remove(filepath.Base(fullPath))
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
