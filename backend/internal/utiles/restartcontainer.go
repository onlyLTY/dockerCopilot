package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RestartContainer(ctx *svc.ServiceContext, id string) error {
	return RestartContainerWithContext(context.Background(), ctx, id)
}

func RestartContainerWithContext(taskCtx context.Context, ctx *svc.ServiceContext, id string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	timeout := 10
	signal := "SIGINT"
	return ctx.DockerClient.ContainerRestart(taskCtx, id, container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	})
}
