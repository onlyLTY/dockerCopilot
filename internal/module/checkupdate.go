package module

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	ref "github.com/distribution/reference"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ImageCheckList struct {
	NeedUpdate bool
}

type ImageUpdateData struct {
	mu      sync.RWMutex
	checkMu sync.Mutex
	data    map[string]ImageCheckList
}

const ContentDigestHeader = "Docker-Content-Digest"

var manifestHTTPClient = secureRegistryHTTPClient(30 * time.Second)

func NewImageCheck() *ImageUpdateData {
	return &ImageUpdateData{data: map[string]ImageCheckList{}}
}

func (i *ImageUpdateData) CheckUpdate(ctx context.Context, dockerClient *client.Client, imageList []types.Image) {
	if !i.checkMu.TryLock() {
		logx.Info("镜像更新检查仍在运行，跳过本轮重复任务")
		return
	}
	defer i.checkMu.Unlock()
	checked := make(map[string]ImageCheckList)
	liveReferences := make(map[string]struct{})
	for _, image := range expandImageReferences(imageList) {
		key := imageref.CacheKey(image.Reference)
		liveReferences[key] = struct{}{}
		parsedReference, err := imageref.ParseTagged(image.Reference)
		if err == nil && parsedReference.Repository == "docker.io/0nlylty/dockercopilot" {
			continue
		}
		needUpdate, comparable := checkSingleImage(ctx, dockerClient, image)
		if comparable {
			checked[key] = ImageCheckList{NeedUpdate: needUpdate}
		}
	}

	i.mu.Lock()
	for reference, previous := range i.data {
		if _, live := liveReferences[reference]; !live {
			continue
		}
		if _, refreshed := checked[reference]; !refreshed {
			checked[reference] = previous
		}
	}
	i.data = checked
	i.mu.Unlock()
}

func expandImageReferences(imageList []types.Image) []types.Image {
	expanded := make([]types.Image, 0, len(imageList))
	for _, image := range imageList {
		references := image.RepoTags
		if len(references) == 0 && image.Reference != "" {
			references = []string{image.Reference}
		}
		for _, value := range references {
			parsed, err := imageref.ParseTagged(value)
			if err != nil {
				if isDigestOnlyReference(value) {
					logx.Debugf("跳过 digest 固定镜像引用 %q：未配置 tag，无法检查更新", value)
				} else {
					logx.Errorf("跳过无法解析的镜像引用 %q: %v", value, err)
				}
				continue
			}
			copy := image
			copy.Reference = parsed.Normalized
			copy.ImageName = parsed.Familiar
			copy.ImageTag = parsed.Tag
			expanded = append(expanded, copy)
		}
	}
	return expanded
}

func isDigestOnlyReference(value string) bool {
	parsed, err := ref.ParseNormalizedNamed(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	_, hasDigest := parsed.(ref.Digested)
	_, hasTag := parsed.(ref.NamedTagged)
	return hasDigest && !hasTag
}

func checkSingleImage(ctx context.Context, dockerClient *client.Client, image types.Image) (bool, bool) {
	imageReference, err := referenceForImage(image)
	if err != nil {
		logx.Errorf("镜像引用无效: %v", err)
		return false, false
	}
	image.Reference = imageReference
	localDigests := repoDigestsForReference(image.RepoDigests, image.Reference)
	if len(localDigests) == 0 {
		logx.Errorf("镜像 %s 没有可比较的本地 RepoDigest", image.Reference)
		return false, false
	}
	remoteDigest, err := getRemoteDigest(ctx, dockerClient, image)
	if err != nil {
		logx.Errorf("获取镜像 %s 的远端 digest 失败: %v", image.Reference, err)
		return false, false
	}
	needUpdate, comparable := compareRepoDigests(localDigests, remoteDigest)
	if !comparable {
		return false, false
	}
	if needUpdate {
		logx.Infof("镜像 %s 有更新，本地 %v，远端 %s", image.Reference, localDigests, remoteDigest)
	}
	return needUpdate, true
}

func getRemoteDigest(ctx context.Context, dockerClient *client.Client, image types.Image) (string, error) {
	imageReference, err := referenceForImage(image)
	if err != nil {
		return "", err
	}
	credentials, credentialErr := credentialsForReference(imageReference)
	if credentialErr != nil {
		logx.Errorf("读取 registry 凭据失败: %v", credentialErr)
	}
	if dockerClient != nil {
		inspectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		distribution, inspectErr := dockerClient.DistributionInspect(inspectCtx, imageReference, credentials.Encoded)
		cancel()
		if inspectErr == nil && distribution.Descriptor.Digest.String() != "" {
			return distribution.Descriptor.Digest.String(), nil
		}
		if inspectErr != nil {
			logx.Errorf("通过 Docker 守护进程获取 %s digest 失败，回退到 Registry API: %v", imageReference, inspectErr)
		}
	}

	token, tokenErr := GetToken(ctx, image, credentials.Basic)
	if tokenErr != nil {
		return "", tokenErr
	}
	digestURL, err := BuildManifestURL(image)
	if err != nil {
		return "", err
	}
	return GetDigest(ctx, digestURL, token)
}

func referenceForImage(image types.Image) (string, error) {
	value := strings.TrimSpace(image.Reference)
	if value == "" && image.ImageName != "" && image.ImageTag != "" {
		value = image.ImageName + ":" + image.ImageTag
	}
	parsed, err := imageref.ParseTagged(value)
	if err != nil {
		return "", err
	}
	return parsed.Normalized, nil
}

func repoDigestsForReference(repoDigests []string, imageReference string) []string {
	parsedTarget, err := imageref.ParseTagged(imageReference)
	if err != nil {
		return nil
	}
	matching := make([]string, 0, len(repoDigests))
	for _, value := range repoDigests {
		named, err := ref.ParseNormalizedNamed(value)
		if err != nil || ref.TrimNamed(named).Name() != parsedTarget.Repository {
			continue
		}
		if _, ok := named.(ref.Digested); ok {
			matching = append(matching, value)
		}
	}
	return matching
}

func compareRepoDigests(repoDigests []string, remoteDigest string) (needUpdate bool, comparable bool) {
	remoteDigest = strings.TrimSpace(remoteDigest)
	if remoteDigest == "" {
		return false, false
	}
	hasLocalDigest := false
	for _, repoDigest := range repoDigests {
		_, localDigest, found := strings.Cut(repoDigest, "@")
		localDigest = strings.TrimSpace(localDigest)
		if !found || localDigest == "" {
			continue
		}
		hasLocalDigest = true
		if localDigest == remoteDigest {
			return false, true
		}
	}
	if !hasLocalDigest {
		return false, false
	}
	return true, true
}

func (i *ImageUpdateData) NeedUpdate(imageReference string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	result, ok := i.data[imageref.CacheKey(imageReference)]
	return ok && result.NeedUpdate
}

func (i *ImageUpdateData) MarkCurrent(imageReference string) {
	key := imageref.CacheKey(imageReference)
	if key == "" {
		return
	}
	i.mu.Lock()
	if i.data == nil {
		i.data = make(map[string]ImageCheckList)
	}
	i.data[key] = ImageCheckList{NeedUpdate: false}
	i.mu.Unlock()
}

func BuildManifestURL(image types.Image) (string, error) {
	imageReference, err := referenceForImage(image)
	if err != nil {
		return "", err
	}
	normalizedRef, err := ref.ParseDockerRef(imageReference)
	if err != nil {
		return "", err
	}
	normalizedTaggedRef, isTagged := normalizedRef.(ref.NamedTagged)
	if !isTagged {
		return "", errors.New("镜像引用没有 tag")
	}
	host, err := GetRegistryAddress(normalizedTaggedRef.Name())
	if err != nil {
		return "", err
	}
	manifestURL := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", ref.Path(normalizedTaggedRef), normalizedTaggedRef.Tag()),
	}
	return manifestURL.String(), nil
}

func GetDigest(ctx context.Context, manifestURL, token string) (string, error) {
	request := func(method string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, manifestURL, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		req.Header.Set("Accept", strings.Join([]string{
			"application/vnd.docker.distribution.manifest.v2+json",
			"application/vnd.docker.distribution.manifest.list.v2+json",
			"application/vnd.oci.image.index.v1+json",
			"application/vnd.oci.image.manifest.v1+json",
		}, ", "))
		return manifestHTTPClient.Do(req)
	}
	response, err := request(http.MethodHead)
	if err != nil {
		return "", err
	}
	if response.StatusCode == http.StatusMethodNotAllowed {
		_ = response.Body.Close()
		response, err = request(http.MethodGet)
		if err != nil {
			return "", err
		}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return "", fmt.Errorf("registry manifest 请求返回 %s", response.Status)
	}
	digest := strings.TrimSpace(response.Header.Get(ContentDigestHeader))
	if digest == "" {
		return "", errors.New("registry 响应缺少 Docker-Content-Digest")
	}
	return digest, nil
}
