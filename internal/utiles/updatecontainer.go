package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
)

func UpdateContainer(ctx *svc.ServiceContext, id string, name string, imageNameAndTag string, delOldContainer bool, taskID string) error {
	ctx.UpdateProgress(taskID, svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Name:       name,
		Message:    "正在连接Docker",
		DetailMsg:  "正在连接Docker",
		IsDone:     false,
	})
	var oldTaskProgress, result = ctx.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		oldTaskProgress.Message = "连接Docker失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	timeout := 10
	signal := "SIGINT"

	ctx.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在拉取新镜像"
	oldTaskProgress.Percentage = 10
	oldTaskProgress.DetailMsg = "正在拉取新镜像"
	ctx.UpdateProgress(taskID, oldTaskProgress)
	cli.NegotiateAPIVersion(context.TODO())
	if err != nil {
		oldTaskProgress.Message = "获取Docker API版本失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	reader, err := cli.ImagePull(context.TODO(), imageNameAndTag, dockerTypes.ImagePullOptions{})
	if err != nil {
		oldTaskProgress.Message = "拉取镜像失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		logx.Errorf("Failed to pull image: %s", err)
	}
	decodePullResp(reader, ctx, taskID)
	oldTaskProgress, result = ctx.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	oldTaskProgress.Message = "拉取镜像成功"
	oldTaskProgress.DetailMsg = "拉取镜像成功"

	oldTaskProgress.Percentage = 30
	oldTaskProgress.Message = "正在停止容器"
	oldTaskProgress.DetailMsg = "正在停止容器"
	ctx.UpdateProgress(taskID, oldTaskProgress)
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err = cli.ContainerStop(context.Background(), id, stopOptions)
	if err != nil {
		oldTaskProgress.Message = "停止容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "容器停止成功"
	oldTaskProgress.DetailMsg = "容器停止成功"

	oldTaskProgress.Percentage = 40
	ctx.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在重命名旧容器"
	oldTaskProgress.DetailMsg = "正在重命名旧容器"
	ctx.UpdateProgress(taskID, oldTaskProgress)

	err = cli.ContainerRename(context.Background(), id, name+"-old")
	if err != nil {
		oldTaskProgress.Message = "重命名旧容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "重命名旧容器成功"
	oldTaskProgress.DetailMsg = "重命名旧容器成功"
	oldTaskProgress.Percentage = 60
	ctx.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在创建新容器"
	oldTaskProgress.DetailMsg = "正在创建新容器"
	ctx.UpdateProgress(taskID, oldTaskProgress)
	inspectedContainer, err := cli.ContainerInspect(context.TODO(), id)
	if err != nil {
		oldTaskProgress.Message = "获取容器信息失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		logx.Error("获取容器信息失败" + err.Error())
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
	_, err = cli.ContainerCreate(context.TODO(), config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		oldTaskProgress.Message = "创建新容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	oldTaskProgress.Message = "创建新容器成功"
	oldTaskProgress.DetailMsg = "创建新容器成功"
	oldTaskProgress.Percentage = 80
	ctx.UpdateProgress(taskID, oldTaskProgress)
	oldTaskProgress.Message = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	oldTaskProgress.DetailMsg = "正在启动新容器以及删除旧容器(如果不保留旧容器)"
	ctx.UpdateProgress(taskID, oldTaskProgress)
	err = cli.ContainerStart(context.Background(), containerName, container.StartOptions{
		CheckpointID:  "",
		CheckpointDir: "",
	})
	if err != nil {
		oldTaskProgress.Message = "启动新容器失败"
		oldTaskProgress.DetailMsg = err.Error()
		oldTaskProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldTaskProgress)
		return err
	}
	if delOldContainer {
		err = cli.ContainerRemove(context.Background(), id, container.RemoveOptions{})
		if err != nil {
			oldTaskProgress.Message = "删除旧容器失败"
			oldTaskProgress.DetailMsg = err.Error()
			oldTaskProgress.IsDone = true
			ctx.UpdateProgress(taskID, oldTaskProgress)
			return err
		}
	}
	oldTaskProgress.Message = "更新成功"
	oldTaskProgress.DetailMsg = "更新成功"
	oldTaskProgress.Percentage = 100
	oldTaskProgress.IsDone = true
	ctx.UpdateProgress(taskID, oldTaskProgress)
	return nil
}

func decodePullResp(reader io.Reader, ctx *svc.ServiceContext, taskID string) {
	decoder := json.NewDecoder(reader)
	var oldTaskProgress, result = ctx.GetProgress(taskID)
	if !result {
		oldTaskProgress = svc.TaskProgress{
			Percentage: 0,
			Name:       "",
			Message:    "",
			DetailMsg:  "",
			IsDone:     false,
		}
	}
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			oldTaskProgress.Message = "拉取镜像失败"
			oldTaskProgress.DetailMsg = err.Error()
			oldTaskProgress.Percentage = 25
			oldTaskProgress.IsDone = true
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Errorf("Failed to decode pull image response: %s", err)
		}
		// Print the progress or error information from the response
		if msg.Error != nil {
			oldTaskProgress.Message = "拉取镜像失败"
			oldTaskProgress.DetailMsg = msg.Error.Error()
			oldTaskProgress.Percentage = 25
			oldTaskProgress.IsDone = true
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Errorf("Error: %s", msg.Error)
		} else {
			var formattedMsg string
			if msg.Progress != nil {
				formattedMsg = fmt.Sprintf("进度%s: %s", msg.Status, msg.Progress.String())
			} else {
				formattedMsg = fmt.Sprintf("进度%s", msg.Status)
			}
			oldTaskProgress.DetailMsg = formattedMsg
			logx.Errorf("Error: %s", formattedMsg)
			oldTaskProgress.Percentage = 25
			ctx.UpdateProgress(taskID, oldTaskProgress)
			logx.Infof("%s: %s\n", msg.Status, msg.Progress)
		}
	}
}
