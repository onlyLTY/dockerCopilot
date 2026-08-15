package utiles

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// RemoveContainer 删除容器。force=true 时先尝试停止再强制删除（运行中也可删）。
func RemoveContainer(ctx *svc.ServiceContext, id string, force bool) error {
	return RemoveContainerWithContext(context.Background(), ctx, id, force)
}

func RemoveContainerWithContext(taskCtx context.Context, ctx *svc.ServiceContext, id string, force bool) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}
	return RemoveContainerWithClient(taskCtx, ctx.DockerClient, id, force, "")
}

// RemoveContainerWithClient 删除容器，并在删除成功后移除对应的更新忽略设置。
// knownName 非空时复用调用方已获取的名称，避免重复 Inspect。
func RemoveContainerWithClient(taskCtx context.Context, dockerClient client.APIClient, id string, force bool, knownName string) error {
	if dockerClient == nil {
		return fmt.Errorf("Docker 客户端不可用")
	}
	if err := taskCtx.Err(); err != nil {
		return err
	}

	name := normalizeContainerName(knownName)
	if name == "" {
		inspect, err := dockerClient.ContainerInspect(taskCtx, id)
		if err != nil {
			return err
		}
		name = normalizeContainerName(inspect.Name)
	}
	if name == "" {
		return fmt.Errorf("容器名称为空：%s", id)
	}

	if force {
		timeout := 10
		if err := dockerClient.ContainerStop(taskCtx, id, container.StopOptions{Timeout: &timeout}); err != nil {
			logx.Errorf("强制删除容器前停止失败 id=%s: %v", id, err)
		}
	}
	if err := dockerClient.ContainerRemove(taskCtx, id, container.RemoveOptions{Force: force}); err != nil {
		return err
	}
	if err := settingstore.SetContainerUpdateIgnored(name, false); err != nil {
		logx.Errorf("删除容器后清理更新忽略设置失败 name=%s: %v", name, err)
	}
	return nil
}

func normalizeContainerName(name string) string {
	return strings.TrimPrefix(strings.TrimSpace(name), "/")
}
