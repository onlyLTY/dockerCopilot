package utiles

import (
	"github.com/docker/docker/api/types/container"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	MyType "github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func GetContainerList(ctx *svc.ServiceContext) ([]MyType.Container, error) {
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	// 获取所有容器（包括停止的容器）
	dockerContainerList, err := ctx.DockerClient.ContainerList(operationContext, container.ListOptions{
		All: true, // 设置为true来获取所有容器
	})
	if err != nil {
		logx.Errorf("get container list error: %v", err)
		return nil, err
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

func CheckImageUpdate(ctx *svc.ServiceContext, containerListData []MyType.Container) []MyType.Container {
	for i, v := range containerListData {
		if ctx.HubImageInfo != nil && ctx.HubImageInfo.NeedUpdate(v.Image) {
			containerListData[i].Update = true
		}
	}
	return containerListData
}
