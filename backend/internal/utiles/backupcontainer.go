package utiles

import (
	"context"
	"encoding/json"
	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"path/filepath"
)

func BackupContainer(ctx *svc.ServiceContext) error {
	containerList, err := GetContainerList(ctx)
	if err != nil {
		return err
	}
	var backupList []dockerBackend.ContainerCreateConfig
	for i, v := range containerList {
		containerID := containerList[i].ID
		ctx.DockerClient.NegotiateAPIVersion(context.TODO())
		inspectedContainer, err := ctx.DockerClient.ContainerInspect(context.TODO(), containerID)
		if err != nil {
			logx.Error("获取容器信息失败" + err.Error())
			return err
		}
		var containerName string
		if len(v.Names) > 0 {
			containerName = v.Names[0][1:]
		} else {
			containerName = "get container name error"
			logx.Error("get container name error" + v.ID)
		}
		inspectedContainer.Config.Hostname = ""
		inspectedContainer.Image = inspectedContainer.Config.Image
		config := inspectedContainer.Config
		hostConfig := inspectedContainer.HostConfig
		var endpoints map[string]*network.EndpointSettings
		if inspectedContainer.NetworkSettings != nil {
			endpoints = inspectedContainer.NetworkSettings.Networks
		}
		networkingConfig := &network.NetworkingConfig{
			EndpointsConfig: endpoints,
		}
		createConfig := dockerBackend.ContainerCreateConfig{Config: config, HostConfig: hostConfig, NetworkingConfig: networkingConfig, Name: containerName}
		backupList = append(backupList, createConfig)
	}
	jsonData, err := json.MarshalIndent(backupList, "", "  ")
	if err != nil {
		logx.Error("Error marshalling data:", err)
		return err
	}
	backupDir := backupstore.Directory()
	fileName, err := backupstore.CreateBackupFile(backupDir, ".json", jsonData, 0644)
	if err != nil {
		logx.Error("Error writing backup file:", err)
		return err
	}
	fullPath := filepath.Join(backupDir, fileName)
	retention, err := backupstore.GetRetention()
	if err != nil {
		logx.Errorf("Error reading backup retention after writing %s: %v", fullPath, err)
		return nil
	}
	if err := backupstore.Retain(retention); err != nil {
		logx.Errorf("Error applying backup retention after writing %s: %v", fullPath, err)
	}
	return nil
}
