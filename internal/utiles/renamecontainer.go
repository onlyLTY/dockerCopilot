package utiles

import (
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	err = ctx.DockerClient.ContainerRename(operationContext, id, newName)
	if err != nil {
		return err
	}
	return nil
}
