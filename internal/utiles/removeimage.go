package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	"log"
)

func RemoveImage(ctx *svc.ServiceContext, imageID string, force bool) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
		return err
	}
	_, err = cli.ImageRemove(context.Background(), imageID, dockerTypes.ImageRemoveOptions{Force: force})
	if err != nil {
		return err
	}
	return nil
}
