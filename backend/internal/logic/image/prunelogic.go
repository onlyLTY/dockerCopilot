package image

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type PruneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPruneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PruneLogic {
	return &PruneLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *PruneLogic) Prune(req *types.ImagePruneReq) (*types.Resp, error) {
	resp := &types.Resp{}
	if req.Kind != "untagged" && req.Kind != "unused" {
		resp.Code = 400
		resp.Msg = "清理类型必须是 untagged 或 unused"
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
		resp.Code = 500
		resp.Msg = fmt.Sprintf("清理镜像失败: %v", err)
		return resp, nil
	}
	deleted := len(report.ImagesDeleted)
	result := map[string]interface{}{"deleted": deleted, "skipped": 0, "errors": []string{}, "spaceReclaimed": report.SpaceReclaimed}
	resp.Code = 200
	resp.Msg = "success"
	resp.Data = result
	return resp, nil
}
