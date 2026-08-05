package utiles

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	MyType "github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func GetContainerList(ctx *svc.ServiceContext) ([]MyType.Container, error) {
	if err := requireDocker(ctx); err != nil {
		return nil, err
	}
	dockerContainerList, err := ctx.DockerClient.ContainerList(context.Background(), container.ListOptions{
		All: true,
	})
	if err != nil {
		logx.Errorf("get container list error: %v", err)
		return nil, err
	}
	containerList := make([]MyType.Container, 0, len(dockerContainerList))
	for _, dockerContainerInfo := range dockerContainerList {
		containerList = append(containerList, MyType.Container{Container: dockerContainerInfo})
	}
	return containerList, nil
}

func CheckImageUpdate(ctx *svc.ServiceContext, containerListData []MyType.Container) []MyType.Container {
	for i, v := range containerListData {
		if result, ok := ctx.HubImageInfo.Get(v.ImageID); ok && result.NeedUpdate {
			containerListData[i].Update = true
		}
	}
	return containerListData
}
