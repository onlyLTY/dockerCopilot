package utiles

import (
	"encoding/json"
	"fmt"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"log"
	"net/http"
	"strings"
)

func GetImagesList(ctx *svc.ServiceContext) ([]types.Image, error) {
	imagelistdata := []types.Image{}
	params := map[string]string{
		"all": "true",
	}
	jwt, endpointsId, err := GetNewJwt(ctx)
	if err != nil {
		return imagelistdata, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/images/json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", jwt)
	query := req.URL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return imagelistdata, err
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&imagelistdata)
	if err != nil {
		return imagelistdata, err
	}
	//看不明白就不要看了，这内存反复的申请，如果你看明白了 给这改成指针吧，啥？我为啥不直接写指针，我懒癌犯了就这样，欢迎pr
	imagelistdata, err = checkImageInUsed(ctx, splitImageNameAndTag(calculateImageSize(imagelistdata)))
	if err != nil {
		return imagelistdata, err
	}
	return imagelistdata, nil
}
func splitImageNameAndTag(imagesList []types.Image) []types.Image {
	for i, image := range imagesList {
		if image.RepoTags != nil {
			imagesList[i].Image_Name = strings.Split(image.RepoTags[0], ":")[0]
			imagesList[i].Image_Tag = strings.Split(image.RepoTags[0], ":")[1]
		} else {
			imagesList[i].Image_Name = strings.Split(image.RepoDigests[0], "@")[0]
			imagesList[i].Image_Tag = "None"
		}
	}
	return imagesList
}
func checkImageInUsed(svc *svc.ServiceContext, imagelist []types.Image) ([]types.Image, error) {
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
