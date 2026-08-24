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
	reader, err := dockerClient.ImagePull(ctx, imageNameAndTag, image.PullOptions{})
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
	wasRunning := inspectedContainer.State != nil && inspectedContainer.State.Running
	clearGeneratedHostname(inspectedContainer.Config, inspectedContainer.ID)
	inspectedContainer.Config.Image = imageNameAndTag
	config := inspectedContainer.Config
	hostConfig := inspectedContainer.HostConfig
	networkingConfig := &network.NetworkingConfig{EndpointsConfig: inspectedContainer.NetworkSettings.Networks}

	stopTimeout := 10
	update(40, "正在停止旧容器", "正在停止旧容器")
	if err := dockerClient.ContainerStop(ctx, id, container.StopOptions{Signal: "SIGINT", Timeout: &stopTimeout}); err != nil {
		return fail("停止容器失败", err)
	}

	backupName := fmt.Sprintf("%s-%s", name, time.Now().Format("2006-01-02-15-04-05.000000000"))
	update(50, "正在保留旧容器", "正在重命名旧容器")
	if err := dockerClient.ContainerRename(ctx, id, backupName); err != nil {
		rollbackErr := runContainerRecovery(func(recoveryCtx context.Context) error {
			return restartOldContainer(recoveryCtx, dockerClient, id, wasRunning)
		})
		return fail("重命名旧容器失败", errors.Join(err, rollbackErr))
	}
	renamed := true
	newContainerID := ""
	rollback := func(cause error) error {
		rollbackErr := runContainerRecovery(func(recoveryCtx context.Context) error {
			return rollbackContainerUpdate(recoveryCtx, dockerClient, id, name, newContainerID, renamed, wasRunning)
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

	update(85, "正在启动新容器", "正在启动新容器")
	if err := dockerClient.ContainerStart(ctx, newContainerID, container.StartOptions{}); err != nil {
		return fail("启动新容器失败，已尝试恢复旧容器", rollback(err))
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

func runContainerRecovery(recoverOperation func(context.Context) error) error {
	recoveryCtx, cancel := context.WithTimeout(context.Background(), containerRecoveryTimeout)
	defer cancel()
	return recoverOperation(recoveryCtx)
}

func rollbackContainerUpdate(ctx context.Context, dockerClient containerUpdateClient, oldID, originalName, newID string, renamed, wasRunning bool) error {
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
	if err := restartOldContainer(ctx, dockerClient, oldID, wasRunning); err != nil {
		rollbackErrors = append(rollbackErrors, err)
	}
	return errors.Join(rollbackErrors...)
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
