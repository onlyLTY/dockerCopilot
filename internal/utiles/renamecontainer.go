package utiles

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	err = cli.ContainerRename(context.TODO(), id, newName)
	if err != nil {
		return err
	}
	return nil
}
