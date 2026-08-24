package utiles

import (
	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func GetContainerInspect(ctx *svc.ServiceContext, id string) (container.InspectResponse, error) {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return container.InspectResponse{}, err
	}
	defer cancel()
	inspectedContainer, err := ctx.DockerClient.ContainerInspect(operationContext, id)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return inspectedContainer, nil
}
