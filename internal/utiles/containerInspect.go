package utiles

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func GetContainerInspect(ctx *svc.ServiceContext, id string) (types.ContainerJSON, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return types.ContainerJSON{}, err
	}
	inspectedContainer, err := cli.ContainerInspect(context.TODO(), id)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return inspectedContainer, nil
}
