package utiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/zeromicro/go-zero/core/logx"
)

const maxBackupFileSize int64 = 128 << 20

type restoreClient interface {
	ImagePull(context.Context, string, image.PullOptions) (io.ReadCloser, error)
	ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, *ocispec.Platform, string) (container.CreateResponse, error)
	ContainerStart(context.Context, string, container.StartOptions) error
	ContainerRemove(context.Context, string, container.RemoveOptions) error
}

func RestoreContainer(serviceContext *svc.ServiceContext, filename, taskID string) error {
	if serviceContext.DockerClient == nil {
		err := errors.New("docker 客户端不可用")
		setRestoreProgress(serviceContext, taskID, 0, "恢复失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), containerUpdateTimeout)
	defer cancel()
	return restoreContainer(ctx, serviceContext, serviceContext.DockerClient, filename, taskID)
}

func restoreContainer(ctx context.Context, serviceContext *svc.ServiceContext, dockerClient restoreClient, filename, taskID string) error {
	setRestoreProgress(serviceContext, taskID, 0, "正在读取备份", "正在读取备份", false, svc.TaskStatusRunning)
	fullPath, err := ResolveBackupPath(filename, ".json")
	if err != nil {
		setRestoreProgress(serviceContext, taskID, 0, "非法文件名", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	content, err := readBackupFile(fullPath)
	if err != nil {
		setRestoreProgress(serviceContext, taskID, 0, "读取备份失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	secret, err := backupEncryptionSecret(serviceContext.Config.Auth.AccessSecret)
	if err != nil {
		setRestoreProgress(serviceContext, taskID, 0, "读取备份失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	plaintext, err := decryptBackup(content, secret)
	if err != nil {
		setRestoreProgress(serviceContext, taskID, 0, "解密备份失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	var configList []containerBackupEntry
	if err := json.Unmarshal(plaintext, &configList); err != nil {
		setRestoreProgress(serviceContext, taskID, 0, "解析备份失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}
	if len(configList) > 500 {
		err := errors.New("单个备份最多恢复 500 个容器")
		setRestoreProgress(serviceContext, taskID, 0, "解析备份失败", err.Error(), true, svc.TaskStatusFailed)
		return err
	}

	results := make([]string, 0, len(configList))
	failureCount := 0
	for index, containerInfo := range configList {
		percentage := 5
		if len(configList) > 0 {
			percentage = 5 + int(float64(index)/float64(len(configList))*90)
		}
		label := containerInfo.Name
		if label == "" {
			label = fmt.Sprintf("第 %d 个容器", index+1)
		}
		setRestoreProgress(serviceContext, taskID, percentage, "正在恢复 "+label, "正在拉取镜像", false, svc.TaskStatusRunning)
		if containerInfo.Config == nil || containerInfo.HostConfig == nil || containerInfo.Config.Image == "" {
			failureCount++
			results = append(results, label+"：备份配置不完整")
			continue
		}

		reader, pullErr := dockerClient.ImagePull(ctx, containerInfo.Config.Image, image.PullOptions{})
		if pullErr != nil {
			failureCount++
			results = append(results, label+"：拉取镜像失败："+pullErr.Error())
			continue
		}
		decodeErr := decodePullResp(reader, serviceContext, taskID)
		closeErr := reader.Close()
		if decodeErr != nil || closeErr != nil {
			failureCount++
			results = append(results, label+"：拉取镜像失败："+errors.Join(decodeErr, closeErr).Error())
			continue
		}

		created, createErr := dockerClient.ContainerCreate(ctx, containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, containerInfo.Platform, containerInfo.Name)
		if createErr != nil {
			failureCount++
			results = append(results, label+"：创建失败："+createErr.Error())
			continue
		}
		if strings.TrimSpace(created.ID) == "" {
			failureCount++
			results = append(results, label+"：创建失败：Docker 未返回容器 ID")
			continue
		}
		if containerInfo.WasRunning != nil && *containerInfo.WasRunning {
			if startErr := dockerClient.ContainerStart(ctx, created.ID, container.StartOptions{}); startErr != nil {
				failureCount++
				results = append(results, label+"：启动失败："+startErr.Error())
				removeErr := runContainerRecovery(func(recoveryCtx context.Context) error {
					return dockerClient.ContainerRemove(recoveryCtx, created.ID, container.RemoveOptions{Force: true})
				})
				if removeErr != nil {
					logx.Errorf("清理启动失败的容器 %s 失败: %v", created.ID, removeErr)
				}
				continue
			}
		}
		results = append(results, label+"：恢复成功")
	}

	detail := strings.Join(results, "\n")
	if failureCount > 0 {
		err := fmt.Errorf("%d/%d 个容器恢复失败", failureCount, len(configList))
		setRestoreProgress(serviceContext, taskID, 100, "恢复部分失败", detail, true, svc.TaskStatusFailed)
		return err
	}
	setRestoreProgress(serviceContext, taskID, 100, "恢复完成", detail, true, svc.TaskStatusCompleted)
	return nil
}

func readBackupFile(path string) ([]byte, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("备份必须是普通文件")
	}
	if info.Size() > maxBackupFileSize {
		return nil, errors.New("备份文件超过大小限制")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || openedInfo.Size() > maxBackupFileSize {
		return nil, errors.New("备份文件无效或超过大小限制")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxBackupFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBackupFileSize {
		return nil, errors.New("备份文件超过大小限制")
	}
	return content, nil
}

func setRestoreProgress(serviceContext *svc.ServiceContext, taskID string, percentage int, message, detail string, done bool, status string) {
	serviceContext.UpdateProgress(taskID, svc.TaskProgress{
		TaskID: taskID, Name: "恢复容器", Percentage: percentage,
		Message: message, DetailMsg: detail, IsDone: done, Status: status,
	})
}
