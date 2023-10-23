package utiles

import (
	"context"
	"encoding/json"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RestoreContainer(ctx *svc.ServiceContext, filename string, taskID string) error {
	var backupList []string
	basePath := `/data/backup` // 指定您的目录
	fullPath := filepath.Join(basePath, filename+".json")

	content, err := os.ReadFile(fullPath)
	if err != nil {
		logx.Error("Failed to read file: %s", err)
	}
	var configList []docker.ContainerCreateConfig
	err = json.Unmarshal(content, &configList)
	if err != nil {
		logx.Error("Failed to parse json: %s", err)
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	for i, containerInfo := range configList {
		info := "正在恢复第" + strconv.Itoa(i+1) + "个容器"
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    info,
		}
		cli.NegotiateAPIVersion(context.TODO())
		if err != nil {
			backupList = append(backupList, "出现错误"+err.Error())
			return err
		}
		reader, err := cli.ImagePull(context.TODO(), containerInfo.Config.Image, docker.ImagePullOptions{})
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
		}
		decodePullResp(reader, ctx, taskID)
		_, err = cli.ContainerCreate(context.TODO(), containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, nil, containerInfo.Name)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 0,
				Message:    info,
			}
			backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
		} else {
			backupList = append(backupList, containerInfo.Name+"恢复成功")
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 0,
				Message:    info,
			}
		}

	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 100,
		Message:    strings.Join(backupList, "\n"),
	}
	return nil
}
