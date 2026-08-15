package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"time"

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
	if !l.svcCtx.TryStartContainerUpdate(req.Id, "direct-rename") {
		return fail(resp, nil, 409, "容器当前正被其他操作占用")
	}
	defer l.svcCtx.FinishContainerUpdate(req.Id, "direct-rename")
	operationCtx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()
	oldName := req.Id
	if inspected, inspectErr := utiles.GetContainerInspectWithContext(operationCtx, l.svcCtx, req.Id); inspectErr == nil && len(inspected.Name) > 1 {
		oldName = inspected.Name[1:]
	}
	err = utiles.RenameContainerWithContext(operationCtx, l.svcCtx, req.Id, req.NewName)
	if err != nil {
		l.Errorf("重命名容器失败 id=%s: %v", req.Id, err)
		return fail(resp, err, 400, "重命名容器失败")
	}
	if migrateErr := settingstore.RenameContainerUpdateIgnore(oldName, req.NewName); migrateErr != nil {
		l.Errorf("迁移容器更新忽略状态失败：%v", migrateErr)
	}
	return ok(resp, nil), nil
}
