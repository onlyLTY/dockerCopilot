package module

import (
	"context"
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

type ImageCheckList struct {
	NeedUpdate bool
}
type ImageUpdateData struct {
	mu   sync.RWMutex
	Data map[string]ImageCheckList
}

const ContentDigestHeader = "Docker-Content-Digest"

func (i *ImageUpdateData) Set(imageID string, value ImageCheckList) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Data[imageID] = value
}

func (i *ImageUpdateData) Get(imageID string) (ImageCheckList, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	value, ok := i.Data[imageID]
	return value, ok
}

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{
		Data: map[string]ImageCheckList{},
	}
}
func (i *ImageUpdateData) CheckUpdate(imageList []types.Image) {
	for _, image := range imageList {
		if strings.Contains(image.ImageName, "0nlylty/dockercopilot") {
			continue
		}
		i.checkSingleImage(image)
	}
}

func (i *ImageUpdateData) CheckUpdateWithProgress(imageList []types.Image, progress func(done, total int, current string)) {
	_ = i.CheckUpdateWithProgressContext(context.Background(), imageList, progress)
}

func (i *ImageUpdateData) CheckUpdateWithProgressContext(ctx context.Context, imageList []types.Image, progress func(done, total int, current string)) error {
	// 排除面板自身镜像后再计总数，进度分母与实际检查一致。
	targets := make([]types.Image, 0, len(imageList))
	for _, image := range imageList {
		if strings.Contains(image.ImageName, "0nlylty/dockercopilot") {
			continue
		}
		targets = append(targets, image)
	}
	total := len(targets)
	for idx, image := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := i.checkSingleImageContext(ctx, image); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		if progress != nil {
			progress(idx+1, total, image.ImageName+":"+image.ImageTag)
		}
	}
	return nil
}

func (i *ImageUpdateData) checkSingleImage(image types.Image) {
	_ = i.checkSingleImageContext(context.Background(), image)
}

func (i *ImageUpdateData) checkSingleImageContext(ctx context.Context, image types.Image) error {
	// 纯本地构建的镜像只有 RepoTags、没有 RepoDigests（RepoDigests 仅在 pull/push
	// 后才会写入）。这类镜像没有可比对的远程引用，向 registry 查询必然 401/404，
	// 因此提前跳过，避免无意义的网络请求与噪音日志。
	if len(image.RepoDigests) == 0 {
		logx.Infof("跳过本地镜像（无远程引用）%s:%s", image.ImageName, image.ImageTag)
		return nil
	}
	token, err := GetTokenWithContext(ctx, image, "")
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logx.Errorf("获取令牌失败，继续检查：%v", err)
	}
	digestURL, err := BuildManifestURLWithContext(ctx, image)
	if err != nil {
		logx.Error("获取digestURL失败" + err.Error())
		return nil
	}
	remoteDigest, err := GetDigestWithContext(ctx, digestURL, token)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// 私有镜像无凭据、镜像已从 registry 删除等均属预期情况，降为 info 避免刷 error。
		logx.Infof("获取digest失败（跳过该镜像更新检查）%s:%s: %v", image.ImageName, image.ImageTag, err)
		return nil
	}
	needUpdate := false
	for _, localRepoDigests := range image.RepoDigests {
		_, localDigest, found := strings.Cut(localRepoDigests, "@")
		if !found {
			continue
		}
		if remoteDigest != localDigest {
			if remoteDigest == "" || localDigest == "" {
				continue
			}
			logx.Info(image.ImageName + ":" + image.ImageTag + " need update")
			logx.Infof("localDigest: %s, remoteDigest: %s", localDigest, remoteDigest)
			needUpdate = true
		} else {
			logx.Info(image.ImageName + ":" + image.ImageTag + " not need update")
		}
	}
	i.Set(image.ID, ImageCheckList{NeedUpdate: needUpdate})
	return nil
}

func BuildManifestURL(image types.Image) (string, error) {
	return BuildManifestURLWithContext(context.Background(), image)
}

func BuildManifestURLWithContext(ctx context.Context, image types.Image) (string, error) {
	normalizedRef, err := ref.ParseDockerRef(image.ImageName + ":" + image.ImageTag)
	if err != nil {
		return "", err
	}
	normalizedTaggedRef, isTagged := normalizedRef.(ref.NamedTagged)
	if !isTagged {
		return "", errors.New("镜像无tag" + normalizedRef.String())
	}

	host, ErrGetRegistryAddress := GetRegistryAddressWithContext(ctx, normalizedTaggedRef.Name())
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
	return GetDigestWithContext(context.Background(), url, token)
}

func GetDigestWithContext(ctx context.Context, url string, token string) (string, error) {
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
		TLSClientConfig:       &tls.Config{},
	}
	client := &http.Client{Transport: tr, Timeout: registryRequestTimeout}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return "", err
	}

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
