package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func StartContainer(ctx *svc.ServiceContext, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	startOptions := dockerTypes.ContainerStartOptions{}
	err = cli.ContainerStart(context.Background(), id, startOptions)
	if err != nil {
		return err
	}

	return nil
}
