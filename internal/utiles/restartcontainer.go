package utiles

import (
	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RestartContainer(ctx *svc.ServiceContext, id string) error {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	timeout := 10
	signal := "SIGINT"
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err = ctx.DockerClient.ContainerRestart(operationContext, id, stopOptions)
	if err != nil {
		return err
	}
	return nil
}
