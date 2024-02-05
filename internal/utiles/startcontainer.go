package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	startOptions := container.StartOptions{}
	err = cli.ContainerStart(context.Background(), id, startOptions)
	if err != nil {
		return err
	}

	return nil
}
