package utiles

import (
	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	startOptions := container.StartOptions{}
	err = ctx.DockerClient.ContainerStart(operationContext, id, startOptions)
	if err != nil {
		return err
	}

	return nil
}
