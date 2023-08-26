package module

import (
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/http"
	"os"
	"strings"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}
type ImageUpdateData struct {
	Data map[string]ImageCheckList
}

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data: map[string]ImageCheckList{},
	}
}
func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	for _, image := range imageList {
		imageName := removeProxy(image.ImageName)
		baseURL := os.Getenv("hubURL")
		if baseURL == "https://hub.docker.com" {
			baseURL = "https://docker.nju.edu.cn"
		}
		if image.ImageTag == "None" {
			logx.Error("镜像tag为空" + image.ImageName + ":" + image.ImageTag)
			continue
		}
		URL := baseURL + "/v2/" + imageName + "/manifests/" + image.ImageTag
		req, err := http.NewRequest("GET", URL, nil)
		if err != nil {
			logx.Error("出现异常" + err.Error() + "URL:" + URL)
			continue
		}
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logx.Error("出现异常" + err.Error() + "URL:" + URL)
			continue
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				panic(err)
			}
		}(resp.Body)

		if resp.StatusCode != 200 {
			logx.Error("获取远程镜像信息失败" + image.ImageName + ":" + image.ImageTag)
			logx.Error("StatusCode:" + resp.Status)
			logx.Error("URL:" + URL)
			continue
		}

		repoDigest := resp.Header.Get("Docker-Content-Digest")
		if repoDigest == "" {
			logx.Error("未从远程获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
			continue
		}
		if len(image.RepoDigests) == 0 {
			logx.Error("未在本地获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
			continue
		}
		localSHA256 := strings.Split(image.RepoDigests[0], "@")[1]
		if repoDigest != localSHA256 {
			if repoDigest == "" || localSHA256 == "" {
				logx.Error("Digest为空" + image.ImageName + ":" + image.ImageTag)
				continue
			}
			logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
			i.Data[image.ID] = ImageCheckList{NeedUpdate: true}
		} else {
			logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
		}
	}

}

func removeProxy(imageName string) string {
	imageNames := strings.Split(imageName, "/")
	if len(imageNames) == 3 {
		//fmt.Println("image_name: " + imageNames[1] + "/" + imageNames[2])
		return imageNames[1] + "/" + imageNames[2]
	} else if len(imageNames) == 2 {
		//fmt.Println("image_name: " + imageNames[0] + "/" + imageNames[1])
		return imageNames[0] + "/" + imageNames[1]
	} else {
		//fmt.Println("image_name: " + imageNames[0])
		return imageNames[0]
	}
}
