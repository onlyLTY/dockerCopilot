package utiles

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	err := ctx.DockerClient.ContainerRename(context.TODO(), id, newName)
	if err != nil {
		return err
	}
	return nil
}
