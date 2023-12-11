package utiles

import (
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	myTypes "github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"
)

func RemoveContainer(ctx *svc.ServiceContext, name string) (myTypes.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return myTypes.MsgResp{}, err
	}
	containerID, err := findContainerIDByName(containers, name)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	// 删除容器
	err = cli.ContainerRemove(context.Background(), containerID, dockerTypes.ContainerRemoveOptions{})
	if err != nil {
		return myTypes.MsgResp{Status: err.Error()}, err
	}

	return myTypes.MsgResp{}, nil
}
