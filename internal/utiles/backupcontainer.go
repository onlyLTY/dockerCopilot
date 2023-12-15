package utiles

import (
	"context"
	"encoding/json"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"path/filepath"
	"time"
)

func BackupContainer(ctx *svc.ServiceContext) ([]string, error) {
	var errList []string
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	containerList, err := GetContainerList(ctx)
	if err != nil {
		return nil, err
	}
	var backerupList []docker.ContainerCreateConfig
	for i, v := range containerList {
		containerID := containerList[i].ID
		cli.NegotiateAPIVersion(context.TODO())
		inspectedContainer, err := cli.ContainerInspect(context.TODO(), containerID)
		if err != nil {
			logx.Error("获取容器信息失败" + err.Error())
			return nil, err
		}
		var containerName, imageNameAndTag string
		if len(v.Names) > 0 {
			containerName = v.Names[0][1:]
		} else {
			containerName = "get container name error"
			logx.Error("get container name error" + v.ID)
		}
		if v.Image != "" {
			imageNameAndTag = v.Image
		} else {
			imageNameAndTag = v.ImageID
			errList = append(errList, containerName+"镜像格式错误，请手动修正")
			logx.Error("image dont have name" + v.ID)
		}
		inspectedContainer.Config.Hostname = ""
		inspectedContainer.Config.Image = imageNameAndTag
		inspectedContainer.Image = imageNameAndTag
		config := inspectedContainer.Config
		hostConfig := inspectedContainer.HostConfig
		networkingConfig := &network.NetworkingConfig{
			EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
		}
		createConfig := docker.ContainerCreateConfig{Config: config, HostConfig: hostConfig, NetworkingConfig: networkingConfig, Name: containerName}
		backerupList = append(backerupList, createConfig)
	}
	jsonData, err := json.MarshalIndent(backerupList, "", "  ")
	if err != nil {
		logx.Error("Error marshalling data:", err)
		return nil, err
	}
	backupDir := os.Getenv("BACKUP_DIR") // 从环境变量中获取备份目录
	if backupDir == "" {
		backupDir = "/data/backup" // 如果环境变量未设置，使用默认值
	}
	_, err = os.Stat(backupDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			logx.Error("Error creating backup directory:", err)
			return nil, err
		}
	}
	currentDate := time.Now().Format("2006-01-02")
	fileName := "backup-" + currentDate + ".json"
	fullPath := filepath.Join(backupDir, fileName)
	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		logx.Error("Error writing to file:", err)
		return nil, err
	}
	return errList, nil
}
