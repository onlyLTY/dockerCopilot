package logic

import (
	"context"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"

	"github.com/zeromicro/go-zero/core/logx"
)

type ImagesListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type imageListItem struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	InUsed     bool   `json:"inUsed"`
	CreateTime string `json:"createTime"`
}

func NewImagesListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImagesListLogic {
	return &ImagesListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ImagesListLogic) ImagesList() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	list, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		l.Errorf("获取镜像列表失败: %v", err)
		if errorx.IsDockerUnavailable(err) {
			resp.Code = errorx.CodeDockerUnavailable
			resp.Msg = "Docker 服务不可用"
		} else {
			resp.Code = 500
			resp.Msg = "获取镜像列表失败"
		}
		resp.Data = map[string]interface{}{}
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	imageInfoList := make([]imageListItem, 0, len(list))
	for _, v := range list {
		imageInfoList = append(imageInfoList, imageListItem{
			Id:         v.ID,
			Name:       v.ImageName,
			Tag:        v.ImageTag,
			Size:       v.SizeFormat,
			InUsed:     v.InUsed,
			CreateTime: time.Unix(v.Created, 0).Format("2006-01-02 15:04:05"),
		})
	}
	resp.Data = imageInfoList
	return resp, nil
}
