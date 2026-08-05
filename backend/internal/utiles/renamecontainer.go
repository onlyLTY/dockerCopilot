package utiles

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	return ctx.DockerClient.ContainerRename(context.Background(), id, newName)
}
