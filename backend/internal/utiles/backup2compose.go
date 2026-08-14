package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	backupCompose "github.com/onlyLTY/dockerCopilot/internal/utiles/backup_compose"
	"github.com/zeromicro/go-zero/core/logx"
)

func Backup2Compose(ctx *svc.ServiceContext) (err error) {
	return Backup2ComposeWithContext(context.Background(), ctx)
}

func Backup2ComposeWithContext(taskCtx context.Context, ctx *svc.ServiceContext) (err error) {
	_, err = Backup2ComposeWithContextResult(taskCtx, ctx)
	return err
}

func Backup2ComposeWithContextResult(taskCtx context.Context, ctx *svc.ServiceContext) (int, error) {
	containerList, err := GetContainerListWithContext(taskCtx, ctx)
	if err != nil {
		return 0, err
	}
	var containerJSONs []dockerTypes.ContainerJSON
	for _, v := range containerList {
		containerID := v.ID
		inspectedContainer, err := GetContainerInspectWithContext(taskCtx, ctx, containerID)
		if err != nil {
			logx.Error("获取容器信息失败" + err.Error())
			return 0, err
		}
		containerJSONs = append(containerJSONs, inspectedContainer)
	}
	err = backupCompose.DockerConfig2ComposeYaml(containerJSONs)
	if err != nil {
		logx.Error("备份失败" + err.Error())
		return 0, err
	}
	return len(containerJSONs), nil
}
