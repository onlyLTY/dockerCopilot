package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

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
	oldName := req.Id
	if l.svcCtx.DockerClient != nil {
		if inspected, inspectErr := l.svcCtx.DockerClient.ContainerInspect(l.ctx, req.Id); inspectErr == nil && len(inspected.Name) > 1 {
			oldName = inspected.Name[1:]
		}
	}
	err = utiles.RenameContainer(l.svcCtx, req.Id, req.NewName)
	if err != nil {
		l.Errorf("重命名容器失败 id=%s: %v", req.Id, err)
		resp.Code = 400
		resp.Msg = "重命名容器失败"
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	if migrateErr := settingstore.RenameContainerUpdateIgnore(oldName, req.NewName); migrateErr != nil {
		l.Errorf("迁移容器更新忽略状态失败：%v", migrateErr)
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{}
	return resp, nil
}
