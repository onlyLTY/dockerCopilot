package image

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/utiles"
	"time"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type Info struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	InUsed     bool   `json:"inUsed"`
	CreateTime string `json:"createTime"`
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	list, err := utiles.GetImagesList(l.svcCtx)
	if err != nil {
		resp.Code = 500
		resp.Msg = err.Error()
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	var imageInfoList []Info
	for _, v := range list {
		var imageInfo Info
		imageInfo.Id = v.ID
		imageInfo.Name = v.ImageName
		imageInfo.Tag = v.ImageTag
		imageInfo.Size = v.SizeFormat
		imageInfo.InUsed = v.InUsed
		t := time.Unix(v.Created, 0)
		imageInfo.CreateTime = t.Format("2006-01-02 15:04:05")
		imageInfoList = append(imageInfoList, imageInfo)
	}
	resp.Data = imageInfoList
	return resp, nil
}
