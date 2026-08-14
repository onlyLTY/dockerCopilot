package module

import (
	"context"
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
	return GetTokenWithContext(context.Background(), image, registryAuth)
}

func GetTokenWithContext(ctx context.Context, image types.Image, registryAuth string) (string, error) {
	logx.Infof("镜像名称 %s", image.ImageName)
	normalizedRef, err := ref.ParseNormalizedNamed(image.ImageName)
	if err != nil {
		return "", fmt.Errorf("解析镜像失败：%s：%w", image.ImageName, err)
	}

	URL := GetChallengeURL(normalizedRef)
	registry := URL.Host

	var req *http.Request
	if req, err = GetChallengeRequestWithContext(ctx, URL); err != nil {
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
		token, err := GetBearerHeaderWithContext(ctx, challenge, normalizedRef, registryAuth)
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
	return GetChallengeRequestWithContext(context.Background(), URL)
}

func GetChallengeRequestWithContext(ctx context.Context, URL url.URL) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Watchtower (Docker)")
	return req, nil
}

func GetBearerHeader(challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	return GetBearerHeaderWithContext(context.Background(), challenge, imageRef, registryAuth)
}

func GetBearerHeaderWithContext(ctx context.Context, challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	client := http.Client{}
	authURL, err := GetAuthURL(challenge, imageRef)

	if err != nil {
		return "", err
	}

	var r *http.Request
	if r, err = http.NewRequestWithContext(ctx, "GET", authURL.String(), nil); err != nil {
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
		// 官方 Hub：并发探测所有候选 host（官方 + 加速源），取最快响应的可用 host。
		// 避免在 index.docker.io 不可达时串行等待每个 host 超时。
		address = quickestHost(DefaultRegistryHost, settingstore.GetHubURLs()...)
	}
	return address, nil
}

// quickestHost 并发探测多个 host，返回第一个可用的；全部不可用时返回 fallback。
func quickestHost(fallback string, hosts ...string) string {
	type result struct {
		host string
		ok   bool
	}
	results := make(chan result, len(hosts)+1)
	all := append([]string{fallback}, hosts...)
	for _, host := range all {
		host := host
		go func() {
			results <- result{host: host, ok: checkHost(host)}
		}()
	}
	for range all {
		if r := <-results; r.ok {
			return r.host
		}
	}
	return fallback
}

// checkHost 用短超时 GET /v2/ 探测 registry 是否可达；200/401 均视为通（未鉴权也正常）。
func checkHost(host string) bool {
	URL := "https://" + host + "/v2/"
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(URL)
	if err != nil {
		// 连接失败是预期行为（网络隔离/防火墙等），降为 Info 避免刷 Error 日志。
		logx.Infof("registry 不可达 %s: %s", URL, err)
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

	logx.Infof("registry 返回非预期状态码 %s: %s", URL, resp.Status)
	return false
}
