package utiles

import (
	"context"

	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RemoveImage(ctx *svc.ServiceContext, imageID string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	_, err := ctx.DockerClient.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: force})
	return err
}
