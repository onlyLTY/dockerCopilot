package utiles

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RenameContainer(ctx *svc.ServiceContext, id string, newName string) error {
	return RenameContainerWithContext(context.Background(), ctx, id, newName)
}

func RenameContainerWithContext(taskCtx context.Context, ctx *svc.ServiceContext, id string, newName string) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	return ctx.DockerClient.ContainerRename(taskCtx, id, newName)
}
