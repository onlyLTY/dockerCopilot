package container

import (
	"context"
	"fmt"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackupLogic {
	return &BackupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BackupLogic) Backup() (resp *types.Resp, err error) {
	return l.submitBackupTask("创建 JSON 容器备份", func(taskCtx context.Context) (backupResult, error) {
		count, err := utiles.BackupContainerWithContextResult(taskCtx, l.svcCtx)
		return backupResult{containerCount: count, detail: fmt.Sprintf("备份完成：创建 1 个 JSON 文件，包含 %d 个容器", count)}, err
	})
}
