package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
)

func StopContainer(ctx *svc.ServiceContext, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	timeout := 10
	signal := "SIGINT"
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err = cli.ContainerStop(context.Background(), id, stopOptions)
	if err != nil {
		return err
	}
	return nil
}
