package utiles

import (
	"context"
	"encoding/json"
	"errors"

	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type containerBackupEntry struct {
	dockerBackend.ContainerCreateConfig
	WasRunning *bool `json:"WasRunning,omitempty"`
	WasPaused  *bool `json:"WasPaused,omitempty"`
}

func BackupContainer(ctx *svc.ServiceContext) error {
	if ctx.DockerClient == nil {
		return errors.New("docker 客户端不可用")
	}
	containerList, err := GetContainerList(ctx)
	if err != nil {
		return err
	}
	backupList := make([]containerBackupEntry, 0, len(containerList))
	for _, listedContainer := range containerList {
		inspectContext, cancel := context.WithTimeout(context.Background(), dockerAPITimeout)
		inspectedContainer, err := ctx.DockerClient.ContainerInspect(inspectContext, listedContainer.ID)
		cancel()
		if err != nil {
			return err
		}
		if inspectedContainer.Config == nil || inspectedContainer.HostConfig == nil || inspectedContainer.NetworkSettings == nil {
			return errors.New("docker 返回的容器配置不完整")
		}
		containerName := listedContainer.ID
		if len(listedContainer.Names) > 0 {
			containerName = listedContainer.Names[0]
			if len(containerName) > 0 && containerName[0] == '/' {
				containerName = containerName[1:]
			}
		}
		clearGeneratedHostname(inspectedContainer.Config, inspectedContainer.ID)
		wasPaused := inspectedContainer.State != nil && inspectedContainer.State.Paused
		wasRunning := inspectedContainer.State != nil && (inspectedContainer.State.Running || wasPaused)
		backupList = append(backupList, containerBackupEntry{
			ContainerCreateConfig: dockerBackend.ContainerCreateConfig{
				Config:           inspectedContainer.Config,
				HostConfig:       inspectedContainer.HostConfig,
				NetworkingConfig: networkingConfigForRecreate(inspectedContainer),
				Name:             containerName,
			},
			WasRunning: &wasRunning,
			WasPaused:  &wasPaused,
		})
	}
	jsonData, err := json.MarshalIndent(backupList, "", "  ")
	if err != nil {
		return err
	}
	secret, err := backupEncryptionSecret(ctx.Config.Auth.AccessSecret)
	if err != nil {
		return err
	}
	encrypted, err := encryptBackup(jsonData, secret)
	if err != nil {
		return err
	}
	backupDir, err := ensureBackupDir()
	if err != nil {
		return err
	}
	filename := newBackupFilename(".json")
	if err := writeBackupAtomic(backupDir, filename, encrypted); err != nil {
		logx.Errorf("写入备份失败: %v", err)
		return err
	}
	return nil
}
