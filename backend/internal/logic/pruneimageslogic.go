package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PruneImagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPruneImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PruneImagesLogic {
	return &PruneImagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PruneImagesLogic) PruneImages(req *types.ImagePruneReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if req.Kind != "untagged" && req.Kind != "unused" {
		resp.Code = 400
		resp.Msg = "清理类型必须是 untagged 或 unused"
		return resp, nil
	}
	return l.prune(req.Kind)
}

func (l *PruneImagesLogic) SubmitPrune(req *types.ImagePruneReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if req.Kind != "untagged" && req.Kind != "unused" {
		resp.Code = 400
		resp.Msg = "清理类型必须是 untagged 或 unused"
		return resp, nil
	}
	if err := l.svcCtx.RequireDocker(); err != nil {
		resp.Code = errorx.CodeDockerUnavailable
		resp.Msg = "Docker 服务不可用"
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	taskID := uuid.NewString()
	name := "清理镜像"
	l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Refresh: true, Name: name, Message: "任务已提交"})
	taskCtx := l.svcCtx.RegisterTask(taskID)
	go func() {
		defer l.svcCtx.FinishTask(taskID)
		defer func() {
			if recovered := recover(); recovered != nil {
				l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "清理失败", DetailMsg: fmt.Sprintf("清理任务异常：%v", recovered), Failed: true, IsDone: true})
			}
		}()
		result, err := pruneImagesWithContext(taskCtx, l.svcCtx, req.Kind)
		if err != nil {
			if taskCtx.Err() != nil {
				l.svcCtx.MarkTaskCanceled(taskID, "任务已停止；已执行的镜像清理不会自动回滚")
				return
			}
			l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "清理失败", DetailMsg: "清理镜像失败：" + err.Error(), Failed: true, IsDone: true})
			return
		}
		l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "清理完成", DetailMsg: formatPruneDetail(result), IsDone: true})
	}()
	resp.Code, resp.Msg, resp.Data = 200, "success", map[string]interface{}{"taskID": taskID}
	return resp, nil
}

func formatReclaimedSize(bytes uint64) string {
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

type pruneResult struct {
	deleted        int
	spaceReclaimed uint64
	images         []prunedImage
}

type prunedImage struct {
	reference string
	id        string
	size      uint64
}

func formatPruneDetail(result pruneResult) string {
	lines := []string{fmt.Sprintf("删除 %d 个镜像，实际回收 %s", result.deleted, formatReclaimedSize(result.spaceReclaimed))}
	for _, item := range result.images {
		lines = append(lines, fmt.Sprintf("- %s（镜像大小 %s）", item.reference, formatReclaimedSize(item.size)))
	}
	return strings.Join(lines, "\n")
}

func (l *PruneImagesLogic) prune(kind string) (*types.Resp, error) {
	resp := &types.Resp{}
	result, err := pruneImagesWithContext(l.ctx, l.svcCtx, kind)
	if err != nil {
		l.Errorf("清理镜像失败 kind=%s: %v", kind, err)
		if errorx.IsDockerUnavailable(err) {
			resp.Code, resp.Msg = errorx.CodeDockerUnavailable, "Docker 服务不可用"
		} else {
			resp.Code, resp.Msg = 500, "清理镜像失败"
		}
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	resp.Code, resp.Msg = 200, "success"
	resp.Data = map[string]interface{}{"deleted": result.deleted, "skipped": 0, "errors": []string{}, "spaceReclaimed": result.spaceReclaimed}
	return resp, nil
}

func pruneImagesWithContext(ctx context.Context, svcCtx *svc.ServiceContext, kind string) (pruneResult, error) {
	if err := svcCtx.RequireDocker(); err != nil {
		return pruneResult{}, err
	}
	if !svcCtx.AcquireImageOp(ctx) {
		if err := ctx.Err(); err != nil {
			return pruneResult{}, err
		}
		return pruneResult{}, context.Canceled
	}
	defer svcCtx.ReleaseImageOp()
	before, err := svcCtx.DockerClient.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return pruneResult{}, err
	}
	pruneFilters := filters.NewArgs()
	if kind == "untagged" {
		pruneFilters.Add("dangling", "true")
	} else {
		pruneFilters.Add("dangling", "false")
	}
	report, err := svcCtx.DockerClient.ImagesPrune(ctx, pruneFilters)
	if err != nil {
		return pruneResult{}, err
	}
	after, err := svcCtx.DockerClient.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return pruneResult{}, err
	}
	removed := removedImages(before, after)
	return pruneResult{deleted: len(removed), spaceReclaimed: report.SpaceReclaimed, images: removed}, nil
}

func countRemovedImages(before, after []image.Summary) int {
	return len(removedImages(before, after))
}

func removedImages(before, after []image.Summary) []prunedImage {
	afterIDs := make(map[string]struct{}, len(after))
	for _, item := range after {
		afterIDs[item.ID] = struct{}{}
	}
	removed := make([]prunedImage, 0)
	for _, item := range before {
		if _, exists := afterIDs[item.ID]; !exists {
			removed = append(removed, prunedImage{
				reference: imageReference(item),
				id:        item.ID,
				size:      uint64(maxInt64(item.Size)),
			})
		}
	}
	return removed
}

func imageReference(item image.Summary) string {
	for _, tag := range item.RepoTags {
		if tag != "" && tag != "<none>:<none>" {
			return tag
		}
	}
	for _, digest := range item.RepoDigests {
		if digest != "" {
			return digest
		}
	}
	id := item.ID
	if strings.HasPrefix(id, "sha256:") {
		id = strings.TrimPrefix(id, "sha256:")
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

func maxInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
