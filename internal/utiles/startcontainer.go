package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	myTypes "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
)

func StartContainer(ctx *svc.ServiceContext, name string) (myTypes.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return myTypes.MsgResp{}, err
	}
	containerID, err := findContainerIDByName(containers, name)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	startOptions := dockerTypes.ContainerStartOptions{}
	err = cli.ContainerStart(context.Background(), containerID, startOptions)
	if err != nil {
		return myTypes.MsgResp{Msg: err.Error()}, err
	}

	return myTypes.MsgResp{}, nil
}
