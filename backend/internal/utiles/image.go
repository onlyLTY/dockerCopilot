package utiles

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"strings"
)

func GetImagesList(ctx *svc.ServiceContext) ([]types.Image, error) {
	if err := requireDocker(ctx); err != nil {
		return nil, err
	}
	var imagesList []types.Image
	dockerImages, err := ctx.DockerClient.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, img := range dockerImages {
		i := types.Image{
			Summary:    img,
			ImageName:  "",
			ImageTag:   "",
			InUsed:     false,
			SizeFormat: "",
		}
		imagesList = append(imagesList, i)
	}
	// 链式多遍遍历会反复分配切片；体量通常不大，保持值传递可读性优先。
	imagesList, err = checkImageInUsed(ctx, splitImageNameAndTag(calculateImageSize(imagesList)))
	if err != nil {
		return imagesList, err
	}
	return imagesList, nil
}

func splitImageNameAndTag(imagesList []types.Image) []types.Image {
	for i, imageInfo := range imagesList {
		if len(imageInfo.RepoTags) != 0 {
			ref := imageInfo.RepoTags[0]
			if index := strings.LastIndex(ref, ":"); index > strings.LastIndex(ref, "/") {
				imagesList[i].ImageName = ref[:index]
				imagesList[i].ImageTag = ref[index+1:]
			} else {
				imagesList[i].ImageName = ref
				imagesList[i].ImageTag = "latest"
			}
		} else if len(imageInfo.RepoDigests) != 0 {
			imagesList[i].ImageName = strings.Split(imageInfo.RepoDigests[0], "@")[0]
			imagesList[i].ImageTag = "None"
		} else {
			imagesList[i].ImageName = "None"
			imagesList[i].ImageTag = "None"
		}
	}
	return imagesList
}
func checkImageInUsed(svc *svc.ServiceContext, imageList []types.Image) ([]types.Image, error) {
	list, err := GetContainerList(svc)
	if err != nil {
		return imageList, err
	}
	for _, v := range list {
		for i, imageInfo := range imageList {
			if v.ImageID == imageInfo.ID {
				imageList[i].InUsed = true
				break
			}
		}
	}
	return imageList, nil
}
func calculateImageSize(imagesList []types.Image) []types.Image {
	for i := range imagesList {
		if imagesList[i].Size >= 1024*1024*1024 {
			imagesList[i].SizeFormat = // Convert size to gigabytes
				fmt.Sprintf("%d Gb", imagesList[i].Size/1024/1024/1024)
		} else {
			imagesList[i].SizeFormat = // Convert size to megabytes
				fmt.Sprintf("%d Mb", imagesList[i].Size/1024/1024)
		}
	}
	return imagesList
}
