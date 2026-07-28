package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	startOptions := container.StartOptions{}
	err := ctx.DockerClient.ContainerStart(context.Background(), id, startOptions)
	if err != nil {
		return err
	}

	return nil
}
