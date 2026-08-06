package module

import (
	"fmt"
	"strings"

	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
)

// ResolvePullCandidates 为项目内 ImagePull 生成候选引用列表。
//
// 规则：
//   - 非 docker.io 镜像（ghcr / 私有仓等）：原样返回，不改写
//   - docker.io / 无前缀官方镜像：
//     1. 若 hubUrls 非空：按顺序生成 mirror/path:tag（或 @digest）
//     2. 最后附带原始引用作为回退
//
// localName 为建议的本地 tag 目标（通常即调用方传入的原始引用），
// pull 成功后应 ImageTag 回该名，避免容器 Config.Image 变成加速域名。
func ResolvePullCandidates(imageRef string) (candidates []string, localName string, err error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return nil, "", fmt.Errorf("镜像引用为空")
	}
	localName = imageRef

	named, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		// 解析失败时不阻塞 pull：交给 Docker 自己报错
		return []string{imageRef}, imageRef, nil
	}

	// 与 Docker 习惯一致：无 tag/digest 时默认 latest
	named = ref.TagNameOnly(named)

	domain := ref.Domain(named)
	if domain != DefaultRegistryDomain {
		return []string{localName}, localName, nil
	}

	path := ref.Path(named)
	var suffix string
	switch n := named.(type) {
	case ref.Canonical:
		suffix = "@" + n.Digest().String()
	case ref.NamedTagged:
		suffix = ":" + n.Tag()
	default:
		suffix = ":latest"
	}

	seen := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		candidates = append(candidates, s)
	}

	for _, host := range settingstore.GetHubURLs() {
		host = strings.TrimSpace(host)
		if host == "" || strings.EqualFold(host, DefaultRegistryDomain) || strings.EqualFold(host, DefaultRegistryHost) {
			continue
		}
		add(host + "/" + path + suffix)
	}
	// 原始引用始终作为最后回退（官方或 daemon mirror）
	add(localName)
	if familiar := ref.FamiliarString(named); familiar != localName {
		add(familiar)
	}
	if full := named.String(); full != localName {
		add(full)
	}

	if len(candidates) == 0 {
		candidates = []string{localName}
	}
	return candidates, localName, nil
}
