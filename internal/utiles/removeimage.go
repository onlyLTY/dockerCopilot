package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	myTypes "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"log"
)

func RemoveImage(ctx *svc.ServiceContext, imageID string, force bool) (myTypes.MsgResp, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
	}
	_, err = cli.ImageRemove(context.Background(), imageID, dockerTypes.ImageRemoveOptions{Force: force})
	if err != nil {
		return myTypes.MsgResp{Msg: err.Error()}, err
	}

	return myTypes.MsgResp{}, nil
}
