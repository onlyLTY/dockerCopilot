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

// PullProgress describes the latest status and aggregate progress of an image pull.
type PullProgress struct {
	Candidate  string
	Status     string
	LayerID    string
	Current    int64
	Total      int64
	Percentage int
}

// PullImage 按 hubUrls 对官方 Docker Hub 镜像尝试加速拉取，并在成功后把本地镜像
// tag 回用户原始引用（如 nginx:latest），避免容器 Config.Image 变成加速域名。
//
// onProgress 可选；用于更新任务进度文案。返回实际拉取成功的引用。
func PullImage(ctx context.Context, cli client.APIClient, imageRef string, onProgress func(string)) (pulledRef string, err error) {
	return pullImageWithProgress(ctx, cli, imageRef, func(progress PullProgress) {
		if onProgress == nil {
			return
		}
		message := progress.Status
		if message == "" {
			message = "正在拉取镜像"
		}
		onProgress(fmt.Sprintf("%s：%s", message, progress.Candidate))
	})
}

func pullImageWithProgress(ctx context.Context, cli client.APIClient, imageRef string, onProgress func(PullProgress)) (pulledRef string, err error) {
	if cli == nil {
		return "", fmt.Errorf("Docker 客户端不可用")
	}
	candidates, localName, err := module.ResolvePullCandidates(imageRef)
	if err != nil {
		return "", err
	}
	if onProgress == nil {
		onProgress = func(PullProgress) {}
	}

	var errs []string
	for i, candidate := range candidates {
		if ctx == nil {
			return "", fmt.Errorf("拉取镜像上下文为空")
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if len(candidates) > 1 {
			onProgress(PullProgress{Candidate: fmt.Sprintf("[%d/%d] %s", i+1, len(candidates), candidate), Status: "正在连接镜像源"})
		} else {
			onProgress(PullProgress{Candidate: candidate, Status: "正在连接镜像源"})
		}
		logx.Infof("ImagePull try %s (local %s)", candidate, localName)

		reader, pullErr := cli.ImagePull(ctx, candidate, image.PullOptions{})
		if pullErr != nil {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, pullErr))
			logx.Infof("ImagePull start failed %s: %v", candidate, pullErr)
			continue
		}
		drainErr := drainPullStreamWithProgress(reader, candidate, onProgress)
		_ = reader.Close()
		if drainErr != nil {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, drainErr))
			logx.Infof("ImagePull stream failed %s: %v", candidate, drainErr)
			continue
		}

		// 加速源拉下来的名字可能是 mirror/library/nginx:latest；
		// tag 回 localName，容器创建仍用用户视角的镜像名。
		if candidate != localName {
			if tagErr := cli.ImageTag(ctx, candidate, localName); tagErr != nil {
				if err := ctx.Err(); err != nil {
					return "", err
				}
				errs = append(errs, fmt.Sprintf("%s: 为 %s 创建本地标签失败: %v", candidate, localName, tagErr))
				logx.Errorf("ImageTag %s -> %s failed: %v", candidate, localName, tagErr)
				continue
			}
			logx.Infof("ImageTag %s -> %s ok", candidate, localName)
		}
		onProgress(PullProgress{Candidate: candidate, Status: "拉取镜像成功", Percentage: 100})
		return localName, nil
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("无可用源")
	}
	return "", fmt.Errorf("%s", strings.Join(errs, "；"))
}

// PullImageWithTask 供更新/恢复等带 task 进度的场景使用。
func PullImageWithTask(ctx context.Context, svcCtx *svc.ServiceContext, imageRef, taskID string) error {
	return PullImageWithTaskRange(ctx, svcCtx, imageRef, taskID, 10, 30)
}

// PullImageWithTaskRange 将镜像拉取进度直接写入任务进度条。
// start/end 参数保留用于兼容已有调用，拉取阶段不再映射任务总进度区间。
func PullImageWithTaskRange(ctx context.Context, svcCtx *svc.ServiceContext, imageRef, taskID string, start, end int) error {
	if svcCtx == nil || svcCtx.DockerClient == nil {
		return fmt.Errorf("Docker 客户端不可用")
	}
	_, err := pullImageWithProgress(ctx, svcCtx.DockerClient, imageRef, func(p PullProgress) {
		progress, ok := svcCtx.GetProgress(taskID)
		if !ok {
			progress = svc.TaskProgress{TaskID: taskID}
		}
		progress.Message = "正在拉取新镜像"
		progress.DetailMsg = formatPullProgress(p)
		progress.Percentage = progress.Percentage
		progress.StepPercentage = p.Percentage

		svcCtx.UpdateProgress(taskID, progress)
	})
	return err
}

// PullImageWithProgress 拉取镜像并通过结构化回调报告 Docker layer 聚合进度。
func PullImageWithProgress(ctx context.Context, cli client.APIClient, imageRef string, onProgress func(PullProgress)) (string, error) {
	return pullImageWithProgress(ctx, cli, imageRef, onProgress)
}

// FormatPullProgress 将拉取状态格式化为任务详情文案。
func FormatPullProgress(progress PullProgress) string {
	status := progress.Status
	if status == "" {
		status = "正在拉取镜像"
	}
	size := ""
	if progress.Total > 0 {
		size = fmt.Sprintf(" · %s/%s", formatPullBytes(progress.Current), formatPullBytes(progress.Total))
	}
	if progress.Candidate != "" && progress.Percentage > 0 {
		return fmt.Sprintf("%s · %s · 拉取进度 %d%%%s", status, progress.Candidate, progress.Percentage, size)
	}
	if progress.Candidate != "" {
		return fmt.Sprintf("%s · %s%s", status, progress.Candidate, size)
	}
	return status + size
}

func formatPullProgress(progress PullProgress) string {
	return FormatPullProgress(progress)
}

func formatPullBytes(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func pullTaskPercentage(pullPercentage, start, end int) int {
	if pullPercentage < 0 {
		pullPercentage = 0
	}
	if pullPercentage > 100 {
		pullPercentage = 100
	}
	return start + (end-start)*pullPercentage/100
}

func drainPullStream(reader io.Reader) error {
	return drainPullStreamWithProgress(reader, "", nil)
}

func drainPullStreamWithProgress(reader io.Reader, candidate string, onProgress func(PullProgress)) error {
	decoder := json.NewDecoder(reader)
	layers := make(map[string]pullLayerProgress)
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
		if msg.ErrorMessage != "" {
			return fmt.Errorf("%s", msg.ErrorMessage)
		}
		if onProgress == nil || msg.ID == "" {
			continue
		}
		layer := layers[msg.ID]
		layer.Status = msg.Status
		if msg.Progress != nil {
			layer.Current = msg.Progress.Current
			layer.Total = msg.Progress.Total
		}
		if pullLayerDone(msg.Status) {
			layer.Done = true
		}
		layers[msg.ID] = layer
		current, total := aggregatePullBytes(layers)
		onProgress(PullProgress{
			Candidate:  candidate,
			LayerID:    msg.ID,
			Status:     pullStatus(msg.Status),
			Current:    current,
			Total:      total,
			Percentage: aggregatePullPercentage(layers),
		})

	}
}

type pullLayerProgress struct {
	Status  string
	Current int64
	Total   int64
	Done    bool
}

func pullLayerDone(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "already exists", "pull complete", "complete":
		return true
	default:
		return false
	}
}

func aggregatePullBytes(layers map[string]pullLayerProgress) (current, total int64) {
	for _, layer := range layers {
		if layer.Current > 0 {
			current += layer.Current
		}
		if layer.Total > 0 {
			total += layer.Total
		}
	}
	if total > 0 && current > total {
		current = total
	}
	return current, total
}
func aggregatePullPercentage(layers map[string]pullLayerProgress) int {
	if len(layers) == 0 {
		return 0
	}
	var total float64
	allDone := true
	for _, layer := range layers {
		ratio := 0.0
		switch {
		case layer.Done:
			ratio = 1
		case layer.Total > 0:
			ratio = float64(layer.Current) / float64(layer.Total)
			allDone = false
		default:
			allDone = false
		}
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		total += ratio
	}
	percentage := int(total/float64(len(layers))*100 + 0.5)
	if percentage >= 100 && !allDone {
		return 99
	}
	return percentage
}

func pullStatus(status string) string {
	if status == "" {
		return "正在拉取镜像"
	}
	return status
}
