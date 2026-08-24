package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContainersListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type Info struct {
	Id          string   `json:"id"`
	Status      string   `json:"status"`
	Name        string   `json:"name"`
	UsingImage  string   `json:"usingImage"`
	CreateImage string   `json:"createImage"`
	CreateTime  string   `json:"createTime"`
	RunningTime string   `json:"runningTime"`
	HaveUpdate  bool     `json:"haveUpdate"`
	IsSelf      bool     `json:"isSelf"`
	IconHints   []string `json:"iconHints,omitempty"`
}

func NewContainersListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainersListLogic {
	return &ContainersListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainersListLogic) ContainersList() (resp *types.Resp, err error) {
	// 获取所有容器（包括停止的容器）
	resp = &types.Resp{}
	list, err := utiles.GetContainerList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	resp.Msg = "success"
	resp.Code = 200
	var containerInfoList []Info
	list = utiles.CheckImageUpdate(l.svcCtx, list)
	for _, v := range list {
		var containerInfo Info
		containerInfo.Id = v.ID
		containerInfo.Status = v.State
		if len(v.Names) > 0 && strings.TrimPrefix(v.Names[0], "/") != "" {
			containerInfo.Name = strings.TrimPrefix(v.Names[0], "/")
		} else {
			containerInfo.Name = "get container name error"
			l.Error("get container name error" + v.ID)
		}
		if v.Image != "" {
			containerInfo.UsingImage = v.Image
		} else {
			containerInfo.UsingImage = v.ImageID
			l.Error("image dont have name" + v.ID)
		}
		containerInfo.CreateImage = containerInfo.UsingImage
		t := time.Unix(v.Created, 0)
		containerInfo.CreateTime = t.Format("2006-01-02 15:04:05")
		containerInfo.RunningTime = v.Status
		containerInfo.HaveUpdate = v.Update
		containerInfo.IsSelf = utiles.IsSelfContainerID(v.ID)
		for _, key := range []string{
			"org.opencontainers.image.title",
			"org.opencontainers.image.source",
			"com.docker.compose.service",
		} {
			if hint := strings.TrimSpace(v.Labels[key]); hint != "" {
				containerInfo.IconHints = append(containerInfo.IconHints, hint)
			}
		}
		containerInfoList = append(containerInfoList, containerInfo)
	}
	resp.Data = containerInfoList
	return resp, nil
}
