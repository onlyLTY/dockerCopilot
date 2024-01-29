package module

import (
	"crypto/tls"
	"errors"
	"fmt"
	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"strings"
	"time"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}
type ImageUpdateData struct {
	Data map[string]ImageCheckList
}

const ContentDigestHeader = "Docker-Content-Digest"

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
	token, err := GetToken(image, "")
	if err != nil {
		logx.Error("获取token失败或者无需获取token，继续尝试检查" + err.Error())
	}
	digestURL, err := BuildManifestURL(image)
	if err != nil {
		logx.Error("获取digestURL失败" + err.Error())
		return
	}
	remoteDigest, err := GetDigest(digestURL, token)
	if err != nil {
		logx.Error("获取digest失败" + err.Error())
		return
	}
	if len(image.RepoDigests) == 0 {
		logx.Error("未在本地获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
		return
	}
	for _, localRepoDigests := range image.RepoDigests {
		localDigest := strings.Split(localRepoDigests, "@")[1]
		if remoteDigest != localDigest {
			if remoteDigest == "" || localDigest == "" {
				logx.Error("Digest为空" + image.ImageName + ":" + image.ImageTag)
				continue
			}
			logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
			logx.Infof("localDigest: %s, remoteDigest: %s", localDigest, remoteDigest)
			i.Data[image.ID] = ImageCheckList{NeedUpdate: true}
			return
		} else {
			logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
		}
	}
}

func BuildManifestURL(image types.Image) (string, error) {
	normalizedRef, err := ref.ParseDockerRef(image.ImageName + ":" + image.ImageTag)
	if err != nil {
		return "", err
	}
	normalizedTaggedRef, isTagged := normalizedRef.(ref.NamedTagged)
	if !isTagged {
		return "", errors.New("镜像无tag" + normalizedRef.String())
	}

	host, _ := GetRegistryAddress(normalizedTaggedRef.Name())
	img, tag := ref.Path(normalizedTaggedRef), normalizedTaggedRef.Tag()

	if err != nil {
		return "", err
	}

	url := url2.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", img, tag),
	}
	return url.String(), nil
}

func GetDigest(url string, token string) (string, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequest("HEAD", url, nil)

	if token != "" {
		req.Header.Add("Authorization", token)
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("GetDigest关闭body失败" + err.Error())
		}
	}(res.Body)

	if res.StatusCode != 200 {
		wwwAuthHeader := res.Header.Get("www-authenticate")
		if wwwAuthHeader == "" {
			wwwAuthHeader = "not present"
		}
		return "", fmt.Errorf("registry responded to head request with %q, auth: %q", res.Status, wwwAuthHeader)
	}
	return res.Header.Get(ContentDigestHeader), nil
}
