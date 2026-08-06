package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

// RemoveContainer 删除容器。force=true 时先尝试停止再强制删除（运行中也可删）。
func RemoveContainer(ctx *svc.ServiceContext, id string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	bg := context.Background()
	if force {
		timeout := 10
		_ = ctx.DockerClient.ContainerStop(bg, id, container.StopOptions{Timeout: &timeout})
	}
	return ctx.DockerClient.ContainerRemove(bg, id, container.RemoveOptions{Force: force})
}
