package container

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

type ContainersListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type Info struct {
	Id            string `json:"id"`
	Status        string `json:"status"`
	Name          string `json:"name"`
	UsingImage    string `json:"usingImage"`
	CreateImage   string `json:"createImage"`
	CreateTime    string `json:"createTime"`
	RunningTime   string `json:"runningTime"`
	HaveUpdate    bool   `json:"haveUpdate"`
	UpdateIgnored bool   `json:"updateIgnored"`
}

func NewContainersListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContainersListLogic {
	return &ContainersListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ContainersListLogic) ContainersList() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	list, err := utiles.GetContainerList(l.svcCtx)
	if err != nil {
		l.Errorf("获取容器列表失败: %v", err)
		return fail(resp, err, 500, "获取容器列表失败")
	}
	resp.Msg = "success"
	resp.Code = 200
	list = utiles.CheckImageUpdate(l.svcCtx, list)

	// 有限并发 Inspect，降低 N+1 串行延迟；响应字段与原先一致
	const inspectWorkers = 8
	type inspectResult struct {
		idx         int
		createImage string
	}
	createImages := make([]string, len(list))
	jobs := make(chan int, len(list))
	results := make(chan inspectResult, len(list))
	workers := inspectWorkers
	if workers > len(list) {
		workers = len(list)
	}
	if workers < 1 {
		workers = 1
	}
	for w := 0; w < workers; w++ {
		go func() {
			for i := range jobs {
				createImage := ""
				if list[i].Image != "" {
					createImage = list[i].Image
				} else {
					createImage = list[i].ImageID
				}
				containerInspect, inspectErr := utiles.GetContainerInspect(l.svcCtx, list[i].ID)
				if inspectErr != nil {
					l.Errorf("inspect 容器失败 id=%s: %v", list[i].ID, inspectErr)
				} else if containerInspect.Config != nil && containerInspect.Config.Image != "" {
					createImage = containerInspect.Config.Image
				}
				results <- inspectResult{idx: i, createImage: createImage}
			}
		}()
	}
	for i := range list {
		jobs <- i
	}
	close(jobs)
	for range list {
		r := <-results
		createImages[r.idx] = r.createImage
	}

	containerInfoList := make([]Info, 0, len(list))
	for i, v := range list {
		var containerInfo Info
		containerInfo.Id = v.ID
		containerInfo.Status = v.State
		if len(v.Names) > 0 {
			containerInfo.Name = v.Names[0][1:]
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
		containerInfo.CreateImage = createImages[i]
		if containerInfo.CreateImage == "" {
			containerInfo.CreateImage = containerInfo.UsingImage
		}
		t := time.Unix(v.Created, 0)
		containerInfo.CreateTime = t.Format("2006-01-02 15:04:05")
		containerInfo.RunningTime = v.Status
		containerInfo.UpdateIgnored = settingstore.IsContainerUpdateIgnored(containerInfo.Name)
		containerInfo.HaveUpdate = v.Update && !containerInfo.UpdateIgnored
		containerInfoList = append(containerInfoList, containerInfo)
	}
	resp.Data = containerInfoList
	return resp, nil
}
