package utiles

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetContainerInspect(ctx *svc.ServiceContext, id string) (types.ContainerJSON, error) {
	if err := requireDocker(ctx); err != nil {
		return types.ContainerJSON{}, err
	}
	return ctx.DockerClient.ContainerInspect(context.Background(), id)
}
