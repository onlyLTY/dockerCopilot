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
	DefaultRegistryDomain  = "docker.io"
	DefaultRegistryHost    = "index.docker.io"
	registryRequestTimeout = 20 * time.Second
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

	URL := GetChallengeURLWithContext(ctx, normalizedRef)
	registry := URL.Host

	var req *http.Request
	if req, err = GetChallengeRequestWithContext(ctx, URL); err != nil {
		return "", fmt.Errorf("创建认证请求失败：镜像=%s，仓库=%s：%w", image.ImageName, registry, err)
	}

	client := &http.Client{Timeout: registryRequestTimeout}
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
	client := http.Client{Timeout: registryRequestTimeout}
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
	return GetChallengeURLWithContext(context.Background(), imageRef)
}

func GetChallengeURLWithContext(ctx context.Context, imageRef ref.Named) url.URL {
	host, _ := GetRegistryAddressWithContext(ctx, imageRef.Name())

	return url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/v2/",
	}
}

func GetRegistryAddress(imageRef string) (string, error) {
	return GetRegistryAddressWithContext(context.Background(), imageRef)
}

func GetRegistryAddressWithContext(ctx context.Context, imageRef string) (string, error) {
	normalizedRef, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", err
	}

	address := ref.Domain(normalizedRef)
	if address == DefaultRegistryDomain {
		address = quickestHost(ctx, DefaultRegistryHost, settingstore.GetHubURLs()...)
	}
	return address, nil
}

func quickestHost(ctx context.Context, fallback string, hosts ...string) string {
	type result struct {
		host string
		ok   bool
	}
	all := append([]string{fallback}, hosts...)
	results := make(chan result, len(all))
	for _, host := range all {
		host := host
		go func() {
			results <- result{host: host, ok: checkHost(ctx, host)}
		}()
	}
	for range all {
		select {
		case r := <-results:
			if r.ok {
				return r.host
			}
		case <-ctx.Done():
			return fallback
		}
	}
	return fallback
}

func checkHost(ctx context.Context, host string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	URL := "https://" + host + "/v2/"
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, URL, nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logx.Infof("registry 不可达 %s: %s", URL, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}
