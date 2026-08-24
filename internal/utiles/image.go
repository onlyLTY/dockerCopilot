package utiles

import (
	"fmt"
	ref "github.com/distribution/reference"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	MyType "github.com/onlyLTY/dockerCopilot/internal/types"
)

func GetImagesList(ctx *svc.ServiceContext) ([]MyType.Image, error) {
	var imagesList []MyType.Image
	operationContext, cancel, err := dockerContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	dockerImages, err := ctx.DockerClient.ImageList(operationContext, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Docker 镜像列表失败: %w", err)
	}

	for _, img := range dockerImages {
		i := MyType.Image{
			Summary:    img,
			ImageName:  "",
			ImageTag:   "",
			InUsed:     false,
			SizeFormat: "",
		}
		imagesList = append(imagesList, i)
	}
	//看不明白就不要看了，这内存反复地申请，如果你看明白了 给这改成指针吧，啥？我为啥不直接写指针，我懒癌犯了就这样，欢迎pr
	imagesList, err = checkImageInUsed(ctx, splitImageNameAndTag(calculateImageSize(imagesList)))
	if err != nil {
		return imagesList, err
	}
	return imagesList, nil
}

func splitImageNameAndTag(imagesList []MyType.Image) []MyType.Image {
	for i, imageInfo := range imagesList {
		if len(imageInfo.RepoTags) != 0 {
			parsed, err := imageref.ParseTagged(imageInfo.RepoTags[0])
			if err != nil {
				imagesList[i].ImageName = imageInfo.RepoTags[0]
				imagesList[i].ImageTag = "None"
				continue
			}
			imagesList[i].ImageName = parsed.Familiar
			imagesList[i].ImageTag = parsed.Tag
			imagesList[i].Reference = parsed.Normalized
		} else if len(imageInfo.RepoDigests) != 0 {
			imagesList[i].Reference = imageInfo.RepoDigests[0]
			if named, err := ref.ParseNormalizedNamed(imageInfo.RepoDigests[0]); err == nil {
				imagesList[i].ImageName = ref.FamiliarName(ref.TrimNamed(named))
			} else {
				imagesList[i].ImageName = imageInfo.RepoDigests[0]
			}
			imagesList[i].ImageTag = "None"
		} else {
			imagesList[i].ImageName = "None"
			imagesList[i].ImageTag = "None"
		}
	}
	return imagesList
}
func checkImageInUsed(svc *svc.ServiceContext, imageList []MyType.Image) ([]MyType.Image, error) {
	list, err := GetContainerList(svc)
	if err != nil {
		return imageList, err
	}
	// 这里可以用mapreduce 我懒等pr
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
