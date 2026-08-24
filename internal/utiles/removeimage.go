package utiles

import (
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RemoveImage(ctx *svc.ServiceContext, imageID string, force bool) error {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	_, err = ctx.DockerClient.ImageRemove(operationContext, imageID, image.RemoveOptions{Force: force})
	if err != nil {
		return err
	}
	return nil
}
