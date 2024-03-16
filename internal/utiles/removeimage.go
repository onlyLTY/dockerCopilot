package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
)

func RemoveImage(ctx *svc.ServiceContext, imageID string, force bool) error {
	_, err := ctx.DockerClient.ImageRemove(context.Background(), imageID, dockerTypes.ImageRemoveOptions{Force: force})
	if err != nil {
		return err
	}
	return nil
}
