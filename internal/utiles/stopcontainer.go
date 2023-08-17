package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
)

func StopContainer(ctx *svc.ServiceContext, name string) (types.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	containerID, err := findContainerIDByName(containers, name)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	timeout := 10
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	err = cli.ContainerStop(context.Background(), containerID, stopOptions)
	if err != nil {
		return types.MsgResp{Msg: err.Error()}, err
	}
	return types.MsgResp{}, nil
}
