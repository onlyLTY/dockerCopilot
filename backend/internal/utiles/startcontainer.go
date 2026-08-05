package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	return ctx.DockerClient.ContainerStart(context.Background(), id, container.StartOptions{})
}
