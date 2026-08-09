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

const (
	pullStageConnecting  = "正在连接镜像源"
	pullStageWaiting     = "等待镜像层"
	pullStageDownloading = "正在下载镜像"
	pullStageExtracting  = "正在解压镜像"
)

// PullProgress describes the latest status and aggregate progress of an image pull.
type PullProgress struct {
	Candidate     string
	Detail        string
	Status        string
	LayerID       string
	Current       int64
	Total         int64
	Percentage    int
	Indeterminate bool
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

func reportPullFailure(onProgress func(PullProgress), reason string) {
	if onProgress == nil {
		return
	}
	onProgress(PullProgress{
		Status: "镜像源失败",
		Detail: reason,
	})
}

func reportPullSwitch(onProgress func(PullProgress), attempt, total int, candidate string) {
	if onProgress == nil {
		return
	}
	onProgress(PullProgress{
		Status:    "切换镜像源",
		Candidate: fmt.Sprintf("[%d/%d] %s", attempt, total, candidate),
	})
}

func summarizePullError(err error) string {
	if err == nil {
		return "未知错误"
	}
	text := strings.TrimSpace(err.Error())
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "eof"):
		return "连接中断（EOF）"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"):
		return "请求超时"
	case strings.Contains(lower, "unauthorized"), strings.Contains(lower, "denied"):
		return "访问被拒绝"
	case strings.Contains(lower, "not found"), strings.Contains(lower, "notfound"):
		return "镜像或 layer 不存在"
	default:
		const maxLength = 160
		if len([]rune(text)) > maxLength {
			return string([]rune(text)[:maxLength]) + "…"
		}
		return text
	}
}

func pullFailureReason(operation string, err error) string {
	return fmt.Sprintf("%s：%s", operation, summarizePullError(err))
}

func pullFailureDetail(attempt, total int, operation string, err error) string {
	return fmt.Sprintf("第 %d/%d 个源失败：%s", attempt, total, pullFailureReason(operation, err))
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
		if len(candidates) > 1 && i > 0 {
			reportPullSwitch(onProgress, i+1, len(candidates), candidate)
		}
		onProgress(PullProgress{Candidate: candidate, Status: pullStageConnecting})
		logx.Infof("ImagePull try %s (local %s)", candidate, localName)

		reader, pullErr := cli.ImagePull(ctx, candidate, image.PullOptions{})
		if pullErr != nil {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			reason := pullFailureReason("拉取启动失败", pullErr)
			reportPullFailure(onProgress, reason)
			errs = append(errs, fmt.Sprintf("%s: %s", candidate, pullFailureDetail(i+1, len(candidates), "拉取启动失败", pullErr)))
			logx.Infof("ImagePull start failed %s: %v", candidate, pullErr)
			continue
		}
		drainErr := drainPullStreamWithProgress(reader, candidate, onProgress)
		_ = reader.Close()
		if drainErr != nil {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			reason := pullFailureReason("拉取流失败", drainErr)
			reportPullFailure(onProgress, reason)
			errs = append(errs, fmt.Sprintf("%s: %s", candidate, pullFailureDetail(i+1, len(candidates), "拉取流失败", drainErr)))
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
				reason := pullFailureReason("创建本地标签失败", tagErr)
				reportPullFailure(onProgress, reason)
				errs = append(errs, fmt.Sprintf("%s: %s", candidate, pullFailureDetail(i+1, len(candidates), "创建本地标签失败", tagErr)))
				logx.Errorf("ImageTag %s -> %s failed: %v", candidate, localName, tagErr)
				continue
			}
			logx.Infof("ImageTag %s -> %s ok", candidate, localName)
			if _, removeErr := cli.ImageRemove(ctx, candidate, image.RemoveOptions{}); removeErr != nil {
				logx.Infof("ImageRemove temporary tag %s failed: %v", candidate, removeErr)
			} else {
				logx.Infof("ImageRemove temporary tag %s ok", candidate)
			}
		}
		onProgress(PullProgress{Candidate: candidate, Status: pullStageExtracting, Percentage: 100})
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
		message := p.Status
		progressType := ""
		indeterminate := false
		current := int64(0)
		total := int64(0)
		stepPercentage := 0
		detail := formatPullProgress(p)
		switch p.Status {
		case pullStageDownloading:
			message = "正在拉取镜像"
			progressType = svc.ProgressTypeImagePull
			indeterminate = p.Indeterminate
			current = p.Current
			total = p.Total
			stepPercentage = p.Percentage
		case pullStageExtracting:
			message = "正在拉取镜像"
			progressType = svc.ProgressTypeImagePull
			indeterminate = true
			stepPercentage = 100
			if detail != "" {
				detail = "正在解压镜像 · " + detail
			} else {
				detail = "正在解压镜像"
			}
		}
		progress.Message = message
		progress.DetailMsg = detail
		progress.ProgressType = progressType
		progress.Indeterminate = indeterminate
		progress.Current = current
		progress.Total = total
		progress.StepPercentage = stepPercentage

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
	if progress.Detail != "" {
		if progress.Candidate != "" {
			return fmt.Sprintf("%s · %s", progress.Detail, progress.Candidate)
		}
		return progress.Detail
	}
	if progress.Candidate != "" {
		return progress.Candidate
	}
	return progress.Status
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
	stage := pullStageWaiting
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
		if strings.EqualFold(strings.TrimSpace(msg.Status), "download complete") && layer.Total > 0 {
			layer.Current = layer.Total
		}
		layers[msg.ID] = layer
		if pullLayerAlreadyExists(msg.Status) {
			continue
		}
		current, total := aggregatePullBytes(layers)
		if strings.EqualFold(strings.TrimSpace(msg.Status), "downloading") {
			stage = pullStageDownloading
		}
		percentage := aggregatePullPercentage(layers)
		indeterminate := stage == pullStageDownloading && (!pullProgressHasKnownTotal(layers) || total <= 0)
		if indeterminate {
			current = 0
			total = 0
			percentage = 0
		}
		onProgress(PullProgress{
			Candidate:     candidate,
			LayerID:       msg.ID,
			Status:        stage,
			Current:       current,
			Total:         total,
			Percentage:    percentage,
			Indeterminate: indeterminate,
		})

	}
}

type pullLayerProgress struct {
	Status  string
	Current int64
	Total   int64
	Done    bool
}

func pullStageRank(stage string) int {
	switch stage {
	case pullStageConnecting:
		return 0
	case pullStageWaiting:
		return 1
	case pullStageDownloading:
		return 2
	case pullStageExtracting:
		return 3
	default:
		return 0
	}
}

func pullStageForLayers(layers map[string]pullLayerProgress) string {
	if len(layers) == 0 {
		return pullStageWaiting
	}
	allDownloaded := true
	hasDownloading := false
	for _, layer := range layers {
		status := strings.ToLower(strings.TrimSpace(layer.Status))
		switch status {
		case "downloading":
			hasDownloading = true
			allDownloaded = false
		case "download complete", "extracting", "extract complete", "pull complete", "complete":
		default:
			if !layer.Done {
				allDownloaded = false
			}
		}
	}
	if allDownloaded {
		return pullStageExtracting
	}
	if hasDownloading {
		return pullStageDownloading
	}
	return pullStageWaiting
}

func pullProgressHasKnownTotal(layers map[string]pullLayerProgress) bool {
	for _, layer := range layers {
		if !layer.Done && layer.Total <= 0 {
			return false
		}
	}
	return len(layers) > 0
}

func pullLayerAlreadyExists(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "already exists")
}

func pullLayerDone(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "already exists", "download complete", "pull complete", "complete":
		return true
	default:
		return false
	}
}

func pullStatus(status string) string {
	status = strings.TrimSpace(status)
	switch strings.ToLower(status) {
	case "waiting", "preparing":
		return pullStageWaiting
	case "downloading":
		return pullStageDownloading
	case "download complete", "extracting", "extract complete", "pull complete", "complete":
		return pullStageExtracting
	}
	if status == "" {
		return pullStageWaiting
	}
	return status
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
	current, total := aggregatePullBytes(layers)
	if total <= 0 {
		return 0
	}
	percentage := int(float64(current)/float64(total)*100 + 0.5)
	if percentage >= 100 {
		for _, layer := range layers {
			if !layer.Done {
				return 99
			}
		}
		return 100
	}
	if percentage < 0 {
		return 0
	}
	return percentage
}
