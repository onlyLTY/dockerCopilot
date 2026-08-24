package utiles

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
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
	resolvedImageIDs := currentImageIDsByReference(ctx)
	for i, v := range containerListData {
		if IsSelfContainerID(v.ID) {
			containerListData[i].Update = false
			continue
		}
		remoteUpdate := ctx.HubImageInfo != nil && ctx.HubImageInfo.NeedUpdate(v.Image)
		key := imageref.CacheKey(v.Image)
		containerListData[i].Update = containerNeedsImageUpdate(remoteUpdate, resolvedImageIDs[key], v.ImageID)
	}
	return containerListData
}

// currentImageIDsByReference resolves every local tag with one Docker call.
// This keeps the frequently polled container-list endpoint from issuing one
// ImageInspect request per distinct image.
func currentImageIDsByReference(ctx *svc.ServiceContext) map[string]string {
	resolved := make(map[string]string)
	if ctx == nil || ctx.DockerClient == nil {
		return resolved
	}
	listContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	images, err := ctx.DockerClient.ImageList(listContext, image.ListOptions{})
	if err != nil {
		logx.Debugf("无法列出镜像以解析当前 ImageID: %v", err)
		return resolved
	}
	return imageIDsByReference(images)
}

func imageIDsByReference(images []image.Summary) map[string]string {
	resolved := make(map[string]string)
	for _, localImage := range images {
		for _, tag := range localImage.RepoTags {
			if key := imageref.CacheKey(tag); key != "" {
				resolved[key] = localImage.ID
			}
		}
	}
	return resolved
}

func containerNeedsImageUpdate(remoteUpdate bool, resolvedImageID, containerImageID string) bool {
	if remoteUpdate {
		return true
	}
	resolvedImageID = strings.TrimPrefix(strings.TrimSpace(resolvedImageID), "sha256:")
	containerImageID = strings.TrimPrefix(strings.TrimSpace(containerImageID), "sha256:")
	if resolvedImageID == "" || containerImageID == "" {
		return false
	}
	return resolvedImageID != containerImageID
}
