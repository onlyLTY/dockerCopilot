package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func StopContainer(ctx *svc.ServiceContext, id string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	timeout := 10
	signal := "SIGINT"
	return ctx.DockerClient.ContainerStop(context.Background(), id, container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	})
}
