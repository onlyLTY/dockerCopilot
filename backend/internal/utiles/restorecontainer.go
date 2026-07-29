package utiles

import (
	"context"
	"encoding/json"
	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"strconv"
	"strings"
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
		if err != nil {
			backupList = append(backupList, "出现错误"+err.Error())
			logx.Errorf("Failed to inspect container: %s", err)
			return err
		}
		reader, err := ctx.DockerClient.ImagePull(context.TODO(), containerInfo.Config.Image, image.PullOptions{})
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}
		defer reader.Close()
		err = decodePullResp(reader, ctx, taskID)
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
			continue
		}
		_, err = ctx.DockerClient.ContainerCreate(context.TODO(), containerInfo.Config, containerInfo.HostConfig, containerInfo.NetworkingConfig, nil, containerInfo.Name)
		if err != nil {
			logx.Errorf("Failed to create container: %s", err)
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			continue
		} else {
			backupList = append(backupList, containerInfo.Name+"恢复成功")
		}
	}
	oldProgress.Percentage = 100
	oldProgress.DetailMsg = strings.Join(backupList, ",\n")
	oldProgress.Message = "恢复完成"
	oldProgress.IsDone = true
	ctx.UpdateProgress(taskID, oldProgress)
	return nil
}
