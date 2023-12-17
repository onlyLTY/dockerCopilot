package module

import (
	"encoding/json"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/http"
	"strings"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}
type ImageUpdateData struct {
	Data map[string]ImageCheckList
}
type AuthResp struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data: map[string]ImageCheckList{},
	}
}
func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	for _, image := range imageList {
		i.checkSingleImage(image)
	}
}

func (i *ImageUpdateData) checkSingleImage(image types.Image) {
	imageName := formatImageName(image.ImageName)
	authURL := "https://auth.docker.io/token"
	authResp, err := http.Get(authURL + "?service=registry.docker.io&scope=repository:" + imageName + ":pull")
	if err != nil {
		logx.Error("获取token失败" + err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("关闭Body失败" + err.Error())
		}
	}(authResp.Body)
	if authResp.StatusCode != http.StatusOK {
		logx.Error("GET请求返回状态码: %d %s\n", authResp.StatusCode, authResp.Status)
		return
	}
	body, err := io.ReadAll(authResp.Body)
	if err != nil {
		logx.Error("获取token失败" + err.Error())
		return
	}
	var auth AuthResp
	err = json.Unmarshal(body, &auth)
	if err != nil {
		logx.Error("获取token失败" + err.Error())
		return
	}
	baseURL := "https://registry-1.docker.io/v2/"
	imageDigestReq, err := http.NewRequest("HEAD", baseURL+imageName+"/manifests/"+image.ImageTag, nil)
	if err != nil {
		logx.Error("创建repoDigestReq失败" + err.Error())
		return
	}
	imageDigestReq.Header.Set("Authorization", "Bearer "+auth.AccessToken)
	imageDigestReq.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	imageDigestResp, err := http.DefaultClient.Do(imageDigestReq)
	if err != nil {
		logx.Error("获取repoDigest失败" + err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("关闭Body失败" + err.Error())
		}
	}(imageDigestResp.Body)
	if imageDigestResp.StatusCode != http.StatusOK {
		logx.Errorf("HEAD请求返回状态码: %d %s\n", imageDigestResp.StatusCode, imageDigestResp.Status)
		return
	}
	repoDigest := imageDigestResp.Header.Get("Docker-Content-Digest")
	if repoDigest == "" {
		logx.Error("未从远程获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
		return
	}
	if len(image.RepoDigests) == 0 {
		logx.Error("未在本地获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
		return
	}
	localSHA256 := strings.Split(image.RepoDigests[0], "@")[1]
	if repoDigest != localSHA256 {
		if repoDigest == "" || localSHA256 == "" {
			logx.Error("Digest为空" + image.ImageName + ":" + image.ImageTag)
			return
		}
		logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
		logx.Infof("localDigest: %s, remoteDigest: %s", localSHA256, repoDigest)
		i.Data[image.ID] = ImageCheckList{NeedUpdate: true}
	} else {
		logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
	}
}

func formatImageName(imageName string) string {
	imageNames := strings.Split(imageName, "/")
	if len(imageNames) == 3 {
		//fmt.Println("image_name: " + imageNames[1] + "/" + imageNames[2])
		return imageNames[1] + "/" + imageNames[2]
	} else if len(imageNames) == 2 {
		//fmt.Println("image_name: " + imageNames[0] + "/" + imageNames[1])
		return imageNames[0] + "/" + imageNames[1]
	} else {
		//fmt.Println("image_name: " + imageNames[0])
		return "library/" + imageNames[0]
	}
}
