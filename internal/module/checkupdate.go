package module

import (
	"encoding/json"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"os"
	"strings"
	"time"
)

// 检查更新处理后的镜像列表
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
		imagename := removeProxy(image.ImageName)
		baseURL := os.Getenv("hubURL")
		r, err := http.Get(baseURL + "/v2/repositories/" + imagename +
			"/tags/" + image.ImageTag)
		if err != nil || r.StatusCode != 200 {
			logx.Error("获取远程镜像信息失败" + image.ImageName + ":" + image.ImageTag)
			continue
		}
		defer r.Body.Close()
		hubimage := types.HubImageInfo{}
		err = json.NewDecoder(r.Body).Decode(&hubimage)
		if err != nil {
			logx.Error("解析远程镜像信息失败" + image.ImageName + ":" + image.ImageTag)
			continue
		}
		layout := "2006-01-02T15:04:05.999999Z"
		t, err := time.Parse(layout, hubimage.LastUpdated)
		if err != nil {
			logx.Error("解析远程镜像信息失败" + image.ImageName + ":" + image.ImageTag)
		}
		remoteTime := t.Unix()
		localTime := image.Created
		if localTime > remoteTime {
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
