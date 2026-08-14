package logic

import (
	"context"
	"fmt"

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
		l.svcCtx.UpdateProgress(taskID, svc.TaskProgress{TaskID: taskID, Name: name, Percentage: 100, Message: "清理完成", DetailMsg: fmt.Sprintf("删除 %d 个镜像，回收 %d 字节", result.deleted, result.spaceReclaimed), IsDone: true})
	}()
	resp.Code, resp.Msg, resp.Data = 200, "success", map[string]interface{}{"taskID": taskID}
	return resp, nil
}

type pruneResult struct {
	deleted        int
	spaceReclaimed uint64
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
	return pruneResult{deleted: countRemovedImages(before, after), spaceReclaimed: report.SpaceReclaimed}, nil
}

func countRemovedImages(before, after []image.Summary) int {
	afterIDs := make(map[string]struct{}, len(after))
	for _, item := range after {
		afterIDs[item.ID] = struct{}{}
	}
	removed := 0
	for _, item := range before {
		if _, exists := afterIDs[item.ID]; !exists {
			removed++
		}
	}
	return removed
}
