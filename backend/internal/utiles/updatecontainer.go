package utiles

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func UpdateContainer(serviceContext *svc.ServiceContext, id string, name string, imageNameAndTag string, delOldContainer bool, taskID string) error {
	return UpdateContainerWithContext(context.Background(), serviceContext, id, name, imageNameAndTag, delOldContainer, taskID)
}

func UpdateContainerWithContext(taskCtx context.Context, serviceContext *svc.ServiceContext, id string, name string, imageNameAndTag string, delOldContainer bool, taskID string) error {
	if serviceContext == nil {
		return fmt.Errorf("服务上下文不可用")
	}
	if err := requireDocker(serviceContext); err != nil {
		serviceContext.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: "更新 " + name, Message: "更新失败", DetailMsg: "Docker 客户端不可用", IsDone: true})
		return err
	}
	serviceContext.UpdateProgress(taskID, svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Name:       name,
		Message:    "正在连接Docker",
		DetailMsg:  "正在连接Docker",
		IsDone:     false,
	})
	var oldTaskProgress, result = serviceContext.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	timeout := 10
	signal := "SIGINT"

	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在拉取新镜像"
	oldTaskProgress.Percentage = 10
	oldTaskProgress.DetailMsg = "正在拉取新镜像"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	pullTimeoutSec := settingstore.GetPullTimeoutSec()
	if pullTimeoutSec <= 0 {
		pullTimeoutSec = int(serviceContext.Config.PullTimeoutSec)
	}
	if pullTimeoutSec <= 0 {
		pullTimeoutSec = int(config.DefaultPullTimeoutSec)
	}
	pullCtx, cancelPull := context.WithTimeout(taskCtx, time.Duration(pullTimeoutSec)*time.Second)
	defer cancelPull()
	if err := PullImageWithTask(pullCtx, serviceContext, imageNameAndTag, taskID); err != nil {
		if taskCtx.Err() != nil {
			return taskCtx.Err()
		}
		oldTaskProgress, _ = serviceContext.GetProgress(taskID)
		clearPullProgress(&oldTaskProgress)
		oldTaskProgress.Message = "拉取镜像失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		logx.Errorf("Failed to pull image: %s", err)
		return err
	}
	oldTaskProgress, result = serviceContext.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	clearPullProgress(&oldTaskProgress)
	oldTaskProgress.Message = "拉取镜像成功"
	oldTaskProgress.DetailMsg = "拉取镜像成功"

	oldTaskProgress.Percentage = 30
	oldTaskProgress.Message = "正在停止容器"
	oldTaskProgress.DetailMsg = "正在停止容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err := serviceContext.DockerClient.ContainerStop(taskCtx, id, stopOptions)
	if err != nil {
		oldTaskProgress.Message = "停止容器失败"
		oldTaskProgress.DetailMsg = "停止容器失败"
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "容器停止成功"
	oldTaskProgress.DetailMsg = "容器停止成功"

	oldTaskProgress.Percentage = 40
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在重命名旧容器"
	oldTaskProgress.DetailMsg = "正在重命名旧容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	currentDate := time.Now().Format("2006-01-02-15-04-05")
	err = serviceContext.DockerClient.ContainerRename(taskCtx, id, name+"-"+currentDate)
	if err != nil {
		oldTaskProgress.Message = "重命名旧容器失败"
		oldTaskProgress.DetailMsg = "重命名旧容器失败"
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "重命名旧容器成功"
	oldTaskProgress.DetailMsg = "重命名旧容器成功"
	oldTaskProgress.Percentage = 60
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在创建新容器"
	oldTaskProgress.DetailMsg = "正在创建新容器"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	inspectedContainer, err := serviceContext.DockerClient.ContainerInspect(taskCtx, id)
	if err != nil {
		oldTaskProgress.Message = "获取容器信息失败"
		oldTaskProgress.DetailMsg = "获取容器信息失败"
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		logx.Errorf("获取容器信息失败: %v", err)
		return err
	}
	inspectedContainer.Config.Hostname = ""
	inspectedContainer.Config.Image = imageNameAndTag
	inspectedContainer.Image = imageNameAndTag
	config := inspectedContainer.Config
	hostConfig := inspectedContainer.HostConfig
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
	}
	containerName := name
	_, err = serviceContext.DockerClient.ContainerCreate(taskCtx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		oldTaskProgress.Message = "创建新容器失败"
		oldTaskProgress.DetailMsg = "创建新容器失败"
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "创建新容器成功"
	oldTaskProgress.DetailMsg = "创建新容器成功"
	oldTaskProgress.Percentage = 80
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	oldTaskProgress.DetailMsg = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	err = serviceContext.DockerClient.ContainerStart(taskCtx, containerName, container.StartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})
	if err != nil {
		oldTaskProgress.Message = "启动新容器失败"
		oldTaskProgress.DetailMsg = "启动新容器失败"
		oldTaskProgress.IsDone = true
		serviceContext.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	if delOldContainer {
		err = RemoveContainerWithClient(taskCtx, serviceContext.DockerClient, id, false, name+"-"+currentDate)
		if err != nil {
			oldTaskProgress.Message = "删除旧容器失败"
			oldTaskProgress.DetailMsg = "删除旧容器失败"
			oldTaskProgress.IsDone = true
			serviceContext.UpdateProgress(taskID, oldTaskProgress)
			return err
		}
	}
	oldTaskProgress.Message = "更新成功"
	oldTaskProgress.DetailMsg = "更新成功"
	oldTaskProgress.Percentage = 100
	oldTaskProgress.IsDone = true
	serviceContext.UpdateProgress(taskID, oldTaskProgress)
	return nil
}

func clearPullProgress(progress *svc.TaskProgress) {
	progress.ProgressType = ""
	progress.Indeterminate = false
	progress.Current = 0
	progress.Total = 0
}
