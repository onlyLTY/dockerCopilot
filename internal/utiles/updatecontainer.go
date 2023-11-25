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
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"log"
)

func UpdateContainer(ctx *svc.ServiceContext, id string, name string, imageNameAndTag string, delOldContainer bool, taskID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	timeout := 10
	signal := "SIGINT"
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 0,
		Message:    "正在停止容器",
	}
	stopOptions := container.StopOptions{
		Signal:  signal,
		Timeout: &timeout,
	}
	err = cli.ContainerStop(context.Background(), id, stopOptions)
	if err != nil {
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    "停止容器失败" + err.Error(),
			IsDone:     true,
		}
		return err
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 20,
		Message:    "容器停止成功",
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 20,
		Message:    "正在拉取新镜像",
	}
	cli.NegotiateAPIVersion(context.TODO())
	if err != nil {
		return err
	}
	reader, err := cli.ImagePull(context.TODO(), imageNameAndTag, dockerTypes.ImagePullOptions{})
	if err != nil {
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    "拉取镜像失败" + err.Error(),
			IsDone:     true,
		}
		logx.Errorf("Failed to pull image: %s", err)
	}
	decodePullResp(reader, ctx, taskID)
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 40,
		Message:    "镜像拉取成功",
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 40,
		Message:    "正在重命名旧容器",
	}
	err = cli.ContainerRename(context.Background(), id, name+"-old")
	if err != nil {
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    "重命名旧容器失败" + err.Error(),
			IsDone:     true,
		}
		return err
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 60,
		Message:    "重命名旧容器成功",
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 60,
		Message:    "正在创建新容器",
	}
	inspectedContainer, err := cli.ContainerInspect(context.TODO(), id)
	if err != nil {
		logx.Errorf("获取容器信息失败")
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
	containerName := name
	_, err = cli.ContainerCreate(context.TODO(), config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    "创建新容器失败" + err.Error(),
		}
		return err
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 80,
		Message:    "创建新容器成功",
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 80,
		Message:    "正在启动新容器以及删除旧容器(如果不保留旧容器)",
	}
	err = cli.ContainerStart(context.Background(), containerName, dockerTypes.ContainerStartOptions{})
	if err != nil {
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    "启动新容器失败" + err.Error(),
			IsDone:     true,
		}
		return err
	}
	if delOldContainer {
		err = cli.ContainerRemove(context.Background(), id, dockerTypes.ContainerRemoveOptions{})
		if err != nil {
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 0,
				Message:    "删除旧容器失败" + err.Error(),
				IsDone:     true,
			}
			return err
		}
	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 100,
		Message:    "更新成功",
		IsDone:     true,
	}
	return nil
}

func decodePullResp(reader io.Reader, ctx *svc.ServiceContext, taskID string) {
	decoder := json.NewDecoder(reader)
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 25,
				Message:    "拉取镜像失败" + err.Error(),
			}
			logx.Errorf("Failed to decode pull image response: %s", err)
		}
		// Print the progress or error information from the response
		if msg.Error != nil {
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 25,
				Message:    "拉取镜像失败" + msg.Error.Error(),
			}
			logx.Error("Error: %s", msg.Error)
		} else {
			var formattedMsg string
			if msg.Progress != nil {
				formattedMsg = fmt.Sprintf("进度%s: %s", msg.Status, msg.Progress.String())
			} else {
				formattedMsg = fmt.Sprintf("进度%s", msg.Status)
			}
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 25,
				Message:    formattedMsg,
			}
			logx.Info("%s: %s\n", msg.Status, msg.Progress)
		}
	}
}
