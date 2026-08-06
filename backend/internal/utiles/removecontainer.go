package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// RemoveContainer 删除容器。force=true 时先尝试停止再强制删除（运行中也可删）。
func RemoveContainer(ctx *svc.ServiceContext, id string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	bg := context.Background()
	if force {
		timeout := 10
		if err := ctx.DockerClient.ContainerStop(bg, id, container.StopOptions{Timeout: &timeout}); err != nil {
			logx.Errorf("强制删除容器前停止失败 id=%s: %v", id, err)
		}
	}
	return ctx.DockerClient.ContainerRemove(bg, id, container.RemoveOptions{Force: force})
}
