package container

import (
	"context"
	"fmt"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type Backup2composeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBackup2composeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Backup2composeLogic {
	return &Backup2composeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *Backup2composeLogic) Backup2compose() (resp *types.Resp, err error) {
	return l.submitBackupTask("创建 YAML 容器备份", func(taskCtx context.Context) (backupResult, error) {
		count, err := utiles.Backup2ComposeWithContextResult(taskCtx, l.svcCtx)
		return backupResult{containerCount: count, detail: fmt.Sprintf("备份完成：创建 1 个 YAML 文件，包含 %d 个容器", count)}, err
	})
}
