package module

import (
	"crypto/tls"
	"errors"
	"fmt"
	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"strings"
	"sync"
	"time"
)

// ImageCheckList 检查更新处理后的镜像列表
type ImageCheckList struct {
	NeedUpdate bool
}
type ImageUpdateData struct {
	mu   sync.RWMutex
	Data map[string]ImageCheckList
}

const ContentDigestHeader = "Docker-Content-Digest"

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data: map[string]ImageCheckList{},
	}
}

// NeedUpdate 并发安全地查询某镜像（按镜像 ID）是否需要更新
func (i *ImageUpdateData) NeedUpdate(imageID string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	v, ok := i.Data[imageID]
	return ok && v.NeedUpdate
}

// setAll 并发安全地整体替换数据
func (i *ImageUpdateData) setAll(data map[string]ImageCheckList) {
	i.mu.Lock()
	i.Data = data
	i.mu.Unlock()
}

func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	// 保留本地仍存在的镜像的旧状态（单次查询失败不至于丢状态），
	// 同时清理已不存在镜像的过期条目（避免"阴魂不散"的更新提示）。
	i.mu.RLock()
	exists := make(map[string]bool, len(imageList))
	next := make(map[string]ImageCheckList, len(imageList))
	for _, image := range imageList {
		exists[image.ID] = true
	}
	for id, v := range i.Data {
		if exists[id] {
			next[id] = v
		}
	}
	i.mu.RUnlock()

	for _, image := range imageList {
		if strings.Contains(image.ImageName, "0nlylty/dockercopilot") {
			continue
		}
		if result := i.checkSingleImage(image); result != nil {
			next[image.ID] = *result
		}
	}
	i.setAll(next)
}

// checkSingleImage 返回 nil 表示本次检查失败（保留旧状态，不作判断）
func (i *ImageUpdateData) checkSingleImage(image types.Image) *ImageCheckList {
	token, err := GetToken(image, "")
	if err != nil {
		logx.Error("获取token失败或者无需获取token，继续尝试检查" + err.Error())
	}
	digestURL, err := BuildManifestURL(image)
	if err != nil {
		logx.Error("获取digestURL失败" + err.Error())
		return nil
	}
	remoteDigest, err := GetDigest(digestURL, token)
	if err != nil {
		logx.Error("获取digest失败" + err.Error())
		return nil
	}
	if len(image.RepoDigests) == 0 {
		logx.Error("未在本地获取到repoDigest" + image.ImageName + ":" + image.ImageTag)
		return nil
	}
	// 只与**同一仓库**的本地 digest 比较；任一匹配即视为已是最新。
	// （修复旧逻辑：循环内反复赋值，结果被 RepoDigests 末位元素覆盖导致的误报。）
	needUpdate := true
	compared := false
	for _, localRepoDigest := range image.RepoDigests {
		parts := strings.SplitN(localRepoDigest, "@", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		if normalizeRepoName(parts[0]) != normalizeRepoName(image.ImageName) {
			continue
		}
		compared = true
		if parts[1] == remoteDigest {
			needUpdate = false
			break
		}
	}
	if !compared {
		logx.Error("未找到同仓库的本地 digest，跳过检查 " + image.ImageName + ":" + image.ImageTag)
		return nil
	}
	if needUpdate {
		logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
		logx.Infof("remoteDigest: %s", remoteDigest)
	} else {
		logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
	}
	return &ImageCheckList{NeedUpdate: needUpdate}
}

// normalizeRepoName 抹平仓库名常见写法差异（大小写 / docker.io / library 前缀）
func normalizeRepoName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "docker.io/")
	name = strings.TrimPrefix(name, "index.docker.io/")
	name = strings.TrimPrefix(name, "library/")
	return name
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

	host, ErrGetRegistryAddress := GetRegistryAddress(normalizedTaggedRef.Name())
	img, tag := ref.Path(normalizedTaggedRef), normalizedTaggedRef.Tag()

	if ErrGetRegistryAddress != nil {
		return "", ErrGetRegistryAddress
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
