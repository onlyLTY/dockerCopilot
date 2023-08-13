package utiles

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	MyType "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
)

func GetContainerList(ctx *svc.ServiceContext) ([]MyType.Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	// 获取所有容器（包括停止的容器）
	dockerContainerList, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All: true, // 设置为true来获取所有容器
	})
	if err != nil {
		panic(err)
	}
	var containerList []MyType.Container
	for _, dockerContainerInfo := range dockerContainerList {
		containerInfo := MyType.Container{
			Container: dockerContainerInfo,
		}
		containerList = append(containerList, containerInfo)
	}
	return containerList, nil
}

func CheckImageUpdate(ctx *svc.ServiceContext, containerlistdata []MyType.Container) []MyType.Container {
	for i, v := range containerlistdata {
		if _, ok := ctx.HubImageInfo.Data[v.ImageID]; ok {
			if ctx.HubImageInfo.Data[v.ImageID].NeedUpdate {
				containerlistdata[i].Update = true
			}
		}
	}
	return containerlistdata
}
