package utiles

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dokcerCopilot/UGREEN/internal/svc"
	myTypes "github.com/onlyLTY/dokcerCopilot/UGREEN/internal/types"
	"log"
)

type configWrapper struct {
	*container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

func CreateContainer(ctx *svc.ServiceContext, oldName string, newName string, imageNameAndTag string) (myTypes.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return myTypes.MsgResp{}, err
	}
	containerID, err := findContainerIDByName(containers, oldName)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	inspectedContainer, err := cli.ContainerInspect(context.TODO(), containerID)
	if err != nil {
		log.Println("获取容器信息失败")
		log.Fatal(err)
	}

	inspectedContainer.Config.Hostname = ""
	inspectedContainer.Config.Image = imageNameAndTag
	inspectedContainer.Image = imageNameAndTag

	config := inspectedContainer.Config
	hostConfig := inspectedContainer.HostConfig
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
	}

	containerName := newName

	_, err = cli.ContainerCreate(context.TODO(), config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		return myTypes.MsgResp{Msg: err.Error()}, err
	}
	return myTypes.MsgResp{}, nil
}
