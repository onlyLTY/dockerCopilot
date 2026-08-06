package module

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const ChallengeHeader = "WWW-Authenticate"
const (
	DefaultRegistryDomain = "docker.io"
	DefaultRegistryHost   = "index.docker.io"
)

func GetToken(image types.Image, registryAuth string) (string, error) {
	logx.Infof("镜像名称 %s", image.ImageName)
	normalizedRef, err := ref.ParseNormalizedNamed(image.ImageName)
	if err != nil {
		return "", fmt.Errorf("解析镜像失败：%s：%w", image.ImageName, err)
	}

	URL := GetChallengeURL(normalizedRef)
	registry := URL.Host

	var req *http.Request
	if req, err = GetChallengeRequest(URL); err != nil {
		return "", fmt.Errorf("创建认证请求失败：镜像=%s，仓库=%s：%w", image.ImageName, registry, err)
	}

	client := &http.Client{}
	var res *http.Response
	if res, err = client.Do(req); err != nil {
		return "", fmt.Errorf("请求镜像仓库失败：镜像=%s，仓库=%s：%w", image.ImageName, registry, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("关闭获取令牌响应失败：" + err.Error())
		}
	}(res.Body)
	v := strings.TrimSpace(res.Header.Get(ChallengeHeader))
	if v == "" && res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return "", nil
	}

	challenge := strings.ToLower(v)
	if strings.HasPrefix(challenge, "basic") {
		if registryAuth == "" {
			return "", fmt.Errorf("缺少仓库凭据：镜像=%s，仓库=%s", image.ImageName, registry)
		}

		return fmt.Sprintf("Basic %s", registryAuth), nil
	}
	if strings.HasPrefix(challenge, "bearer") {
		token, err := GetBearerHeader(challenge, normalizedRef, registryAuth)
		if err != nil {
			return "", fmt.Errorf("获取令牌失败：镜像=%s，仓库=%s：%w", image.ImageName, registry, err)
		}
		return token, nil
	}

	challengeType := "未知"
	if fields := strings.Fields(v); len(fields) > 0 {
		switch strings.ToLower(fields[0]) {
		case "basic":
			challengeType = "基础认证"
		case "bearer":
			challengeType = "令牌认证"
		case "digest":
			challengeType = "摘要认证"
		default:
			challengeType = "未知认证"
		}
	}
	return "", fmt.Errorf(
		"仓库认证类型不支持：镜像=%s:%s，仓库=%s，状态码=%d，类型=%s",
		image.ImageName, image.ImageTag, registry, res.StatusCode, challengeType,
	)
}

func GetChallengeRequest(URL url.URL) (*http.Request, error) {
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Watchtower (Docker)")
	return req, nil
}

func GetBearerHeader(challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	client := http.Client{}
	authURL, err := GetAuthURL(challenge, imageRef)

	if err != nil {
		return "", err
	}

	var r *http.Request
	if r, err = http.NewRequest("GET", authURL.String(), nil); err != nil {
		return "", fmt.Errorf("创建令牌请求失败：%w", err)
	}

	if registryAuth != "" {
		logx.Info("私有镜像，无法获取是否有更新")
		r.Header.Add("Authorization", fmt.Sprintf("Basic %s", registryAuth))
	} else {
		logx.Info("未找到仓库凭据")
	}

	var authResponse *http.Response
	if authResponse, err = client.Do(r); err != nil {
		return "", fmt.Errorf("请求令牌服务失败：%w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("关闭认证响应 body 失败：" + err.Error())
		}
	}(authResponse.Body)

	if authResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("认证令牌响应状态码错误：%d", authResponse.StatusCode)
	}

	body, err := io.ReadAll(authResponse.Body)
	if err != nil {
		return "", fmt.Errorf("读取认证响应失败：%w", err)
	}

	tokenResponse := &types.TokenResponse{}
	if err = json.Unmarshal(body, tokenResponse); err != nil {
		return "", fmt.Errorf("解析认证响应失败：%w", err)
	}

	if tokenResponse.Token == "" {
		return "", fmt.Errorf("认证令牌为空")
	}

	return fmt.Sprintf("Bearer %s", tokenResponse.Token), nil
}

func GetAuthURL(challenge string, imageRef ref.Named) (*url.URL, error) {
	loweredChallenge := strings.ToLower(challenge)
	raw := strings.TrimPrefix(loweredChallenge, "bearer")

	pairs := strings.Split(raw, ",")
	values := make(map[string]string, len(pairs))

	for _, pair := range pairs {
		trimmed := strings.Trim(pair, " ")
		if key, val, ok := strings.Cut(trimmed, "="); ok {
			values[key] = strings.Trim(val, `"`)
		}
	}
	if values["realm"] == "" || values["service"] == "" {

		return nil, fmt.Errorf("认证信息缺少必要参数")
	}

	authURL, _ := url.Parse(values["realm"])
	q := authURL.Query()
	q.Add("service", values["service"])

	scopeImage := ref.Path(imageRef)

	scope := fmt.Sprintf("repository:%s:pull", scopeImage)
	q.Add("scope", scope)

	authURL.RawQuery = q.Encode()
	return authURL, nil
}

func GetChallengeURL(imageRef ref.Named) url.URL {
	host, _ := GetRegistryAddress(imageRef.Name())

	URL := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/v2/",
	}
	return URL
}

func GetRegistryAddress(imageRef string) (string, error) {
	normalizedRef, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", err
	}

	address := ref.Domain(normalizedRef)

	if address == DefaultRegistryDomain {
		// 官方 Hub：先试 index.docker.io，不通再按用户可配的 hubUrls（默认内置加速列表）依次探测。
		if checkHost(DefaultRegistryHost) {
			address = DefaultRegistryHost
		} else {
			for _, host := range settingstore.GetHubURLs() {
				if checkHost(host) {
					address = host
					break
				}
			}
		}
		if address == DefaultRegistryDomain {
			address = DefaultRegistryHost
		}
	}
	return address, nil
}

// checkHost 用短超时 GET /v2/ 探测 registry 是否可达；200/401 均视为通（未鉴权也正常）。
func checkHost(host string) bool {
	URL := "https://" + host + "/v2/"
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(URL)
	if err != nil {
		logx.Errorf("Failed to connect to %s: %s", URL, err)
		return false
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Errorf("关闭body失败" + err.Error())
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusUnauthorized {
		return true
	}

	logx.Errorf("Failed to connect to %s: %s", URL, resp.Status)
	return false
}
