package utiles

import (
	"context"

	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func RemoveImage(ctx *svc.ServiceContext, target string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	_, err := ctx.DockerClient.ImageRemove(context.Background(), target, image.RemoveOptions{Force: force})
	return err
}
