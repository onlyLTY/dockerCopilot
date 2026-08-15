package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	return StartContainerWithContext(context.Background(), ctx, id)
}

func StartContainerWithContext(taskCtx context.Context, ctx *svc.ServiceContext, id string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	return ctx.DockerClient.ContainerStart(taskCtx, id, container.StartOptions{})
}
