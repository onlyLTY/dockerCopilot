package container

import (
	"context"
	"os"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"

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

func (l *DelRestoreLogic) DelRestore(req *types.ContainerRestoreReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	fileName := req.Filename
	basePath := os.Getenv("BACKUP_DIR") // 从环境变量中获取备份目录
	if basePath == "" {
		basePath = "/data/backups" // 如果环境变量未设置，使用默认值
	}
	fullPath := basePath + "/" + fileName + ".json"
	err = os.Remove(fullPath)
	if err != nil {
		resp.Code = 400
		resp.Msg = "删除失败"
		return resp, nil
	}
	resp.Code = 200
	resp.Msg = "success"
	return resp, nil
}
