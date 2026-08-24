package utiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	containerUpdateTimeout   = 30 * time.Minute
	containerRecoveryTimeout = 2 * time.Minute
)

type containerUpdateClient interface {
	ImagePull(context.Context, string, image.PullOptions) (io.ReadCloser, error)
	ContainerInspect(context.Context, string) (container.InspectResponse, error)
	ContainerStop(context.Context, string, container.StopOptions) error
	ContainerPause(context.Context, string) error
	ContainerUnpause(context.Context, string) error
	ContainerRename(context.Context, string, string) error
	ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, *ocispec.Platform, string) (container.CreateResponse, error)
	ContainerStart(context.Context, string, container.StartOptions) error
	ContainerRemove(context.Context, string, container.RemoveOptions) error
}

func UpdateContainer(serviceContext *svc.ServiceContext, id, name, imageNameAndTag string, delOldContainer bool, taskID string) error {
	if serviceContext.DockerClient == nil {
		err := errors.New("docker 客户端不可用")
		setContainerTaskFailure(serviceContext, taskID, name, "连接 Docker 失败", err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), containerUpdateTimeout)
	defer cancel()
	return updateContainer(ctx, serviceContext, serviceContext.DockerClient, id, name, imageNameAndTag, delOldContainer, taskID)
}

func updateContainer(ctx context.Context, serviceContext *svc.ServiceContext, dockerClient containerUpdateClient, id, name, imageNameAndTag string, delOldContainer bool, taskID string) error {
	progress := svc.TaskProgress{
		TaskID: taskID, Name: name, Percentage: 0,
		Message: "正在连接 Docker", DetailMsg: "正在连接 Docker",
		Status: svc.TaskStatusRunning,
	}
	update := func(percentage int, message, detail string) {
		progress.Percentage = percentage
		progress.Message = message
		progress.DetailMsg = detail
		serviceContext.UpdateProgress(taskID, progress)
	}
	fail := func(message string, err error) error {
		progress.Message = message
		progress.DetailMsg = err.Error()
		progress.IsDone = true
		progress.Status = svc.TaskStatusFailed
		serviceContext.UpdateProgress(taskID, progress)
		return err
	}
	serviceContext.UpdateProgress(taskID, progress)

	update(10, "正在拉取新镜像", "正在拉取新镜像")
	registryAuth, err := module.RegistryAuthForReference(imageNameAndTag)
	if err != nil {
		return fail("读取镜像仓库凭据失败", err)
	}
	reader, err := dockerClient.ImagePull(ctx, imageNameAndTag, image.PullOptions{RegistryAuth: registryAuth})
	if err != nil {
		return fail("拉取镜像失败", err)
	}
	defer reader.Close()
	if err := decodePullResp(reader, serviceContext, taskID); err != nil {
		return fail("拉取镜像失败", err)
	}

	update(30, "正在读取容器配置", "正在读取容器配置")
	inspectedContainer, err := dockerClient.ContainerInspect(ctx, id)
	if err != nil {
		return fail("获取容器信息失败", err)
	}
	if inspectedContainer.Config == nil || inspectedContainer.HostConfig == nil || inspectedContainer.NetworkSettings == nil {
		return fail("获取容器信息失败", errors.New("docker 返回的容器配置不完整"))
	}
	actualName := strings.TrimPrefix(strings.TrimSpace(inspectedContainer.Name), "/")
	if actualName == "" || actualName != name {
		return fail("容器名称已变化，请刷新后重试", errors.New("请求中的容器名称与 Docker 当前状态不一致"))
	}
	wasPaused := inspectedContainer.State != nil && inspectedContainer.State.Paused
	wasRunning := inspectedContainer.State != nil && (inspectedContainer.State.Running || wasPaused)
	clearGeneratedHostname(inspectedContainer.Config, inspectedContainer.ID)
	inspectedContainer.Config.Image = imageNameAndTag
	config := inspectedContainer.Config
	hostConfig := inspectedContainer.HostConfig
	networkingConfig := networkingConfigForRecreate(inspectedContainer)

	stopTimeout := 10
	if wasRunning {
		if wasPaused {
			update(35, "正在解除旧容器暂停", "需要先解除暂停才能停止容器")
			if err := dockerClient.ContainerUnpause(ctx, id); err != nil {
				return fail("解除旧容器暂停失败", err)
			}
		}
		update(40, "正在停止旧容器", "正在停止旧容器")
		if err := dockerClient.ContainerStop(ctx, id, container.StopOptions{Timeout: &stopTimeout}); err != nil {
			return fail("停止容器失败", err)
		}
	} else {
		update(40, "旧容器已停止", "将保持容器的停止状态")
	}

	backupName := fmt.Sprintf("%s-%s", name, time.Now().Format("2006-01-02-15-04-05.000000000"))
	update(50, "正在保留旧容器", "正在重命名旧容器")
	if err := dockerClient.ContainerRename(ctx, id, backupName); err != nil {
		rollbackErr := runContainerRecovery(func(recoveryCtx context.Context) error {
			return restoreOldContainerState(recoveryCtx, dockerClient, id, wasRunning, wasPaused)
		})
		return fail("重命名旧容器失败", errors.Join(err, rollbackErr))
	}
	renamed := true
	newContainerID := ""
	rollback := func(cause error) error {
		rollbackErr := runContainerRecovery(func(recoveryCtx context.Context) error {
			return rollbackContainerUpdate(recoveryCtx, dockerClient, id, name, newContainerID, renamed, wasRunning, wasPaused)
		})
		return errors.Join(cause, rollbackErr)
	}

	update(65, "正在创建新容器", "正在创建新容器")
	created, err := dockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		return fail("创建新容器失败，已尝试恢复旧容器", rollback(err))
	}
	newContainerID = created.ID
	if newContainerID == "" {
		return fail("创建新容器失败，已尝试恢复旧容器", rollback(errors.New("docker 未返回新容器 ID")))
	}

	if wasRunning {
		update(85, "正在启动新容器", "正在启动新容器")
		if err := dockerClient.ContainerStart(ctx, newContainerID, container.StartOptions{}); err != nil {
			return fail("启动新容器失败，已尝试恢复旧容器", rollback(err))
		}
		if wasPaused {
			update(90, "正在恢复暂停状态", "旧容器在更新前处于暂停状态")
			if err := dockerClient.ContainerPause(ctx, newContainerID); err != nil {
				return fail("暂停新容器失败，已尝试恢复旧容器", rollback(err))
			}
		}
	} else {
		update(85, "新容器保持停止", "旧容器更新前处于停止状态")
	}

	detail := "更新成功"
	if delOldContainer {
		if err := dockerClient.ContainerRemove(ctx, id, container.RemoveOptions{}); err != nil {
			detail = "新容器已启动，但旧容器删除失败: " + err.Error()
			logx.Errorf("新容器已启动，删除旧容器失败: %v", err)
		}
	}
	progress.Message = "更新成功"
	progress.DetailMsg = detail
	progress.Percentage = 100
	progress.IsDone = true
	progress.Status = svc.TaskStatusCompleted
	serviceContext.UpdateProgress(taskID, progress)
	if serviceContext.HubImageInfo != nil {
		serviceContext.HubImageInfo.MarkCurrent(imageNameAndTag)
	}
	return nil
}

func networkingConfigForRecreate(inspected container.InspectResponse) *network.NetworkingConfig {
	result := &network.NetworkingConfig{EndpointsConfig: make(map[string]*network.EndpointSettings)}
	if inspected.NetworkSettings == nil {
		return result
	}
	containerName := strings.TrimPrefix(inspected.Name, "/")
	for networkName, endpoint := range inspected.NetworkSettings.Networks {
		if endpoint == nil {
			continue
		}
		aliases := make([]string, 0, len(endpoint.Aliases))
		for _, alias := range endpoint.Aliases {
			alias = strings.TrimSpace(alias)
			generatedIDAlias := inspected.ID != "" && (strings.HasPrefix(inspected.ID, alias) || strings.HasPrefix(alias, inspected.ID))
			if alias == "" || alias == containerName || generatedIDAlias {
				continue
			}
			aliases = append(aliases, alias)
		}
		var ipamConfig *network.EndpointIPAMConfig
		if endpoint.IPAMConfig != nil {
			ipamConfig = endpoint.IPAMConfig.Copy()
		}
		clean := &network.EndpointSettings{
			IPAMConfig: ipamConfig,
			Links:      append([]string(nil), endpoint.Links...),
			Aliases:    aliases,
			DriverOpts: cloneStringMap(endpoint.DriverOpts),
			GwPriority: endpoint.GwPriority,
		}
		if configuredMAC := legacyConfiguredMAC(inspected.Config); configuredMAC != "" {
			clean.MacAddress = configuredMAC
		}
		result.EndpointsConfig[networkName] = clean
	}
	return result
}

func legacyConfiguredMAC(config *container.Config) string {
	if config == nil {
		return ""
	}
	//lint:ignore SA1019 Preserve MAC addresses from containers created through Docker API < 1.44.
	return config.MacAddress
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func runContainerRecovery(recoverOperation func(context.Context) error) error {
	recoveryCtx, cancel := context.WithTimeout(context.Background(), containerRecoveryTimeout)
	defer cancel()
	return recoverOperation(recoveryCtx)
}

func rollbackContainerUpdate(ctx context.Context, dockerClient containerUpdateClient, oldID, originalName, newID string, renamed, wasRunning, wasPaused bool) error {
	var rollbackErrors []error
	if newID != "" {
		if err := dockerClient.ContainerRemove(ctx, newID, container.RemoveOptions{Force: true}); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("删除失败的新容器: %w", err))
		}
	}
	if renamed {
		if err := dockerClient.ContainerRename(ctx, oldID, originalName); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("恢复旧容器名称: %w", err))
		}
	}
	if err := restoreOldContainerState(ctx, dockerClient, oldID, wasRunning, wasPaused); err != nil {
		rollbackErrors = append(rollbackErrors, err)
	}
	return errors.Join(rollbackErrors...)
}

func restoreOldContainerState(ctx context.Context, dockerClient containerUpdateClient, oldID string, wasRunning, wasPaused bool) error {
	if err := restartOldContainer(ctx, dockerClient, oldID, wasRunning); err != nil {
		return err
	}
	if wasPaused {
		if err := dockerClient.ContainerPause(ctx, oldID); err != nil {
			return fmt.Errorf("恢复旧容器暂停状态: %w", err)
		}
	}
	return nil
}

func restartOldContainer(ctx context.Context, dockerClient containerUpdateClient, oldID string, wasRunning bool) error {
	if !wasRunning {
		return nil
	}
	if err := dockerClient.ContainerStart(ctx, oldID, container.StartOptions{}); err != nil {
		return fmt.Errorf("重新启动旧容器: %w", err)
	}
	return nil
}

func setContainerTaskFailure(serviceContext *svc.ServiceContext, taskID, name, message string, err error) {
	serviceContext.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: name, Message: message, DetailMsg: err.Error(),
		IsDone: true, Status: svc.TaskStatusFailed,
	})
}

func decodePullResp(reader io.Reader, ctx *svc.ServiceContext, taskID string) error {
	decoder := json.NewDecoder(reader)
	progress, exists := ctx.GetProgress(taskID)
	if !exists {
		progress = svc.TaskProgress{TaskID: taskID, Status: svc.TaskStatusRunning}
	}
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("解析拉取镜像响应失败: %w", err)
		}
		if msg.Error != nil {
			return fmt.Errorf("拉取镜像失败: %w", msg.Error)
		}
		formattedMsg := "进度" + msg.Status
		if msg.Progress != nil {
			formattedMsg = fmt.Sprintf("进度%s: %s", msg.Status, msg.Progress.String())
		}
		progress.DetailMsg = formattedMsg
		progress.Percentage = 25
		progress.Status = svc.TaskStatusRunning
		ctx.UpdateProgress(taskID, progress)
	}
}
