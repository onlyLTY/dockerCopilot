package utiles

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func GetContainerInspect(ctx *svc.ServiceContext, id string) (types.ContainerJSON, error) {
	inspectedContainer, err := ctx.DockerClient.ContainerInspect(context.TODO(), id)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return inspectedContainer, nil
}
