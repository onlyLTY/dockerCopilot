package utiles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	dockerMsgType "github.com/docker/docker/pkg/jsonmessage"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// PullImage 按 hubUrls 对官方 Docker Hub 镜像尝试加速拉取，并在成功后把本地镜像
// tag 回用户原始引用（如 nginx:latest），避免容器 Config.Image 变成加速域名。
//
// onProgress 可选；用于更新任务进度文案。返回实际拉取成功的引用。
func PullImage(ctx context.Context, cli client.APIClient, imageRef string, onProgress func(string)) (pulledRef string, err error) {
	if cli == nil {
		return "", fmt.Errorf("Docker 客户端不可用")
	}
	candidates, localName, err := module.ResolvePullCandidates(imageRef)
	if err != nil {
		return "", err
	}
	if onProgress == nil {
		onProgress = func(string) {}
	}

	var errs []string
	for i, candidate := range candidates {
		if len(candidates) > 1 {
			onProgress(fmt.Sprintf("正在拉取镜像（%d/%d）：%s", i+1, len(candidates), candidate))
		} else {
			onProgress("正在拉取镜像：" + candidate)
		}
		logx.Infof("ImagePull try %s (local %s)", candidate, localName)

		reader, pullErr := cli.ImagePull(ctx, candidate, image.PullOptions{})
		if pullErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, pullErr))
			logx.Infof("ImagePull start failed %s: %v", candidate, pullErr)
			continue
		}
		drainErr := drainPullStream(reader)
		_ = reader.Close()
		if drainErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, drainErr))
			logx.Infof("ImagePull stream failed %s: %v", candidate, drainErr)
			continue
		}

		// 加速源拉下来的名字可能是 mirror/library/nginx:latest；
		// tag 回 localName，容器创建仍用用户视角的镜像名。
		if candidate != localName {
			if tagErr := cli.ImageTag(ctx, candidate, localName); tagErr != nil {
				logx.Errorf("ImageTag %s -> %s failed: %v", candidate, localName, tagErr)
			} else {
				logx.Infof("ImageTag %s -> %s ok", candidate, localName)
			}
		}
		onProgress("拉取镜像成功：" + localName)
		return candidate, nil
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("拉取镜像失败：无可用源")
	}
	return "", fmt.Errorf("拉取镜像失败：%s", strings.Join(errs, "；"))
}

// PullImageWithTask 供更新/恢复等带 task 进度的场景使用。
func PullImageWithTask(ctx context.Context, svcCtx *svc.ServiceContext, imageRef, taskID string) error {
	if svcCtx == nil || svcCtx.DockerClient == nil {
		return fmt.Errorf("Docker 客户端不可用")
	}
	_, err := PullImage(ctx, svcCtx.DockerClient, imageRef, func(msg string) {
		progress, ok := svcCtx.GetProgress(taskID)
		if !ok {
			progress = svc.TaskProgress{TaskID: taskID}
		}
		progress.Message = "正在拉取新镜像"
		progress.DetailMsg = msg
		progress.Percentage = 25
		svcCtx.UpdateProgress(taskID, progress)
	})
	return err
}

func drainPullStream(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	for {
		var msg dockerMsgType.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Error != nil {
			return msg.Error
		}
	}
}
