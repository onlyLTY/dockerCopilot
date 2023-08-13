package utiles

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	MyType "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"log"
	"strings"
)

func GetImagesList(ctx *svc.ServiceContext) ([]MyType.Image, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Unable to create docker client: %s", err)
	}
	var imagesList []MyType.Image
	dockerImages, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Fatalf("Unable to fetch docker images: %s", err)
	}

	for _, img := range dockerImages {
		image := MyType.Image{
			Containers:  img.Containers,
			Created:     img.Created,
			ID:          img.ID,
			Labels:      img.Labels,
			ParentID:    img.ParentID,
			RepoDigests: img.RepoDigests,
			RepoTags:    img.RepoTags,
			SharedSize:  img.SharedSize,
			Size:        img.Size,
			VirtualSize: img.VirtualSize,
			ImageName:   "",
			ImageTag:    "",
			States:      0,
			SizeFormat:  "",
		}
		imagesList = append(imagesList, image)
	}
	//看不明白就不要看了，这内存反复地申请，如果你看明白了 给这改成指针吧，啥？我为啥不直接写指针，我懒癌犯了就这样，欢迎pr
	imagesList, err = checkImageInUsed(ctx, splitImageNameAndTag(calculateImageSize(imagesList)))
	if err != nil {
		return imagesList, err
	}
	return imagesList, nil
}

func splitImageNameAndTag(imagesList []MyType.Image) []MyType.Image {
	for i, image := range imagesList {
		if len(image.RepoTags) != 0 {
			imagesList[i].ImageName = strings.Split(image.RepoTags[0], ":")[0]
			imagesList[i].ImageTag = strings.Split(image.RepoTags[0], ":")[1]
		} else if len(image.RepoDigests) != 0 {
			imagesList[i].ImageName = strings.Split(image.RepoDigests[0], "@")[0]
			imagesList[i].ImageTag = "None"
		} else {
			imagesList[i].ImageName = "None"
			imagesList[i].ImageTag = "None"
		}
	}
	return imagesList
}
func checkImageInUsed(svc *svc.ServiceContext, imagelist []MyType.Image) ([]MyType.Image, error) {
	list, err := GetContainerList(svc)
	if err != nil {
		return imagelist, err
	}
	// 这里可以用mapreduce 我懒等pr
	for _, v := range list {
		for i, imagev := range imagelist {
			if v.ImageID == imagev.ID {
				imagelist[i].States = 1
				break
			}
		}
	}
	return imagelist, nil
}
func calculateImageSize(imagesList []MyType.Image) []MyType.Image {
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
