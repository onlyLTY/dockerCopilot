package logic

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
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
	if err := l.svcCtx.RequireDocker(); err != nil {
		resp.Code = errorx.CodeDockerUnavailable
		resp.Msg = "Docker 服务不可用"
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	before, err := l.svcCtx.DockerClient.ImageList(l.ctx, image.ListOptions{})
	if err != nil {
		l.Errorf("获取清理前镜像列表失败 kind=%s: %v", req.Kind, err)
		resp.Code = 500
		resp.Msg = "清理镜像失败"
		return resp, nil
	}
	pruneFilters := filters.NewArgs()
	if req.Kind == "untagged" {
		pruneFilters.Add("dangling", "true")
	} else {
		pruneFilters.Add("dangling", "false")
	}
	report, err := l.svcCtx.DockerClient.ImagesPrune(l.ctx, pruneFilters)
	if err != nil {
		l.Errorf("清理镜像失败 kind=%s: %v", req.Kind, err)
		resp.Code = 500
		resp.Msg = "清理镜像失败"
		return resp, nil
	}
	after, err := l.svcCtx.DockerClient.ImageList(l.ctx, image.ListOptions{})
	if err != nil {
		l.Errorf("获取清理后镜像列表失败 kind=%s: %v", req.Kind, err)
		resp.Code = 500
		resp.Msg = "清理镜像失败"
		return resp, nil
	}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = map[string]interface{}{
		"deleted":        countRemovedImages(before, after),
		"skipped":        0,
		"errors":         []string{},
		"spaceReclaimed": report.SpaceReclaimed,
	}
	return resp, nil
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
