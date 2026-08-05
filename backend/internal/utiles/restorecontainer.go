package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

func RestoreContainer(ctx *svc.ServiceContext, filename string, taskID string) error {
	var backupList []string
	fullPath, err := ResolveBackupPath(filename)
	if err != nil {
		return err
	}
	oldProgress := svc.TaskProgress{
		TaskID:     taskID,
		Percentage: 0,
		Message:    "",
		Name:       "",
		DetailMsg:  "",
		IsDone:     false,
	}
	oldProgress.Name = "恢复容器"
	content, err := os.ReadFile(fullPath)
	if err != nil {
		logx.Errorf("Failed to read file: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "读取文件失败或者未找到文件"
		oldProgress.DetailMsg = err.Error()
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
		return err
	}
	var configList []dockerBackend.ContainerCreateConfig
	err = json.Unmarshal(content, &configList)
	if err != nil {
		logx.Errorf("Failed to parse json: %s", err)
		oldProgress.Percentage = 0
		oldProgress.Message = "解析文件失败"
		oldProgress.DetailMsg = err.Error()
		oldProgress.IsDone = true
		ctx.UpdateProgress(taskID, oldProgress)
		return err
	}
	for i, containerInfo := range configList {
		info := "正在恢复第" + strconv.Itoa(i+1) + "个容器"
		oldProgress.Percentage = int(float64(i) / float64(len(configList)) * 100)
		oldProgress.Message = info
		oldProgress.DetailMsg = info
		ctx.UpdateProgress(taskID, oldProgress)
		ctx.DockerClient.NegotiateAPIVersion(context.TODO())

		name := containerInfo.Name
		if name == "" {
			name = "container-" + strconv.Itoa(i+1)
		}
		if blockErr := validateRestoreHostConfig(name, containerInfo.HostConfig); blockErr != nil {
			logx.Errorf("拒绝高危恢复配置 name=%s: %v", name, blockErr)
			backupList = append(backupList, name+"恢复被拒绝: "+blockErr.Error())
			continue
		}

		if containerInfo.Config == nil || containerInfo.Config.Image == "" {
			backupList = append(backupList, name+"恢复失败: 备份缺少镜像信息")
			continue
		}
		reader, err := ctx.DockerClient.ImagePull(context.TODO(), containerInfo.Config.Image, image.PullOptions{})
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像失败")
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}
		err = decodePullResp(reader, ctx, taskID)
		_ = reader.Close()
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像失败")
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}
		_, err = ctx.DockerClient.ContainerCreate(context.TODO(), containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, nil, containerInfo.Name)
		if err != nil {
			logx.Errorf("Failed to create container: %s", err)
			backupList = append(backupList, name+"恢复失败")
			continue
		}
		backupList = append(backupList, name+"恢复成功")
	}
	oldProgress.Percentage = 100
	oldProgress.DetailMsg = strings.Join(backupList, ",\n")
	oldProgress.Message = "恢复完成"
	oldProgress.IsDone = true
	ctx.UpdateProgress(taskID, oldProgress)
	return nil
}

// validateRestoreHostConfig 拦截备份恢复中的极高危 HostConfig（privileged / docker.sock / 敏感挂载）。
func validateRestoreHostConfig(name string, hc *container.HostConfig) error {
	if hc == nil {
		return nil
	}
	if hc.Privileged {
		return fmt.Errorf("容器 %s 启用了 privileged，已拒绝恢复", name)
	}
	for _, b := range hc.Binds {
		src := bindSource(b)
		if strings.Contains(filepath.ToSlash(src), "docker.sock") {
			return fmt.Errorf("容器 %s 挂载了 Docker socket，已拒绝恢复", name)
		}
		if isSensitiveRestorePath(src) {
			return fmt.Errorf("容器 %s 挂载了敏感宿主机路径，已拒绝恢复", name)
		}
	}
	for _, m := range hc.Mounts {
		src := m.Source
		if strings.Contains(filepath.ToSlash(src), "docker.sock") {
			return fmt.Errorf("容器 %s 挂载了 Docker socket，已拒绝恢复", name)
		}
		if isSensitiveRestorePath(src) {
			return fmt.Errorf("容器 %s 挂载了敏感宿主机路径，已拒绝恢复", name)
		}
	}
	return nil
}

func bindSource(bind string) string {
	// host:container[:mode]
	parts := strings.Split(bind, ":")
	if len(parts) == 0 {
		return bind
	}
	// Windows 盘符 C:\... 会被切坏；Linux 备份为主，取第一段即可
	return parts[0]
}

func isSensitiveRestorePath(path string) bool {
	clean := filepath.Clean(path)
	switch clean {
	case "/", "/etc", "/proc", "/sys", "/var/run", "/var/run/docker.sock":
		return true
	default:
		slash := filepath.ToSlash(clean)
		return strings.HasPrefix(slash, "/etc/") || strings.HasPrefix(slash, "/proc/") || strings.HasPrefix(slash, "/sys/")
	}
}
