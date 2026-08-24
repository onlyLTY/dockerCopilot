package module

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

const ChallengeHeader = "WWW-Authenticate"

const (
	DefaultRegistryDomain = "docker.io"
	DefaultRegistryHost   = "registry-1.docker.io"
	maxRegistryBodySize   = 1 << 20
)

var registryHTTPClient = secureRegistryHTTPClient(15 * time.Second)

func secureRegistryHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return errors.New("拒绝 registry 重定向到非 HTTPS 地址")
			}
			if len(via) >= 10 {
				return errors.New("registry 重定向次数过多")
			}
			return nil
		},
	}
}

func GetToken(ctx context.Context, image types.Image, registryAuth string) (string, error) {
	imageReference, err := referenceForImage(image)
	if err != nil {
		return "", err
	}
	normalizedRef, err := ref.ParseNormalizedNamed(imageReference)
	if err != nil {
		return "", err
	}
	challengeURL, err := GetChallengeURL(normalizedRef)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, challengeURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Docker-Copilot")
	res, err := registryHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("registry challenge 返回 %s", res.Status)
	}
	challenge := strings.TrimSpace(res.Header.Get(ChallengeHeader))
	if challenge == "" {
		if res.StatusCode == http.StatusOK {
			return "", nil
		}
		return "", errors.New("registry 未返回 WWW-Authenticate")
	}
	scheme, _, _ := strings.Cut(challenge, " ")
	switch strings.ToLower(scheme) {
	case "basic":
		if registryAuth == "" {
			return "", errors.New("私有 registry 需要凭据")
		}
		return "Basic " + registryAuth, nil
	case "bearer":
		return GetBearerHeader(ctx, challenge, normalizedRef, registryAuth)
	default:
		return "", fmt.Errorf("不支持的 registry 鉴权方式 %q", scheme)
	}
}

func GetBearerHeader(ctx context.Context, challenge string, imageRef ref.Named, registryAuth string) (string, error) {
	authURL, err := GetAuthURL(challenge, imageRef)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL.String(), nil)
	if err != nil {
		return "", err
	}
	if registryAuth != "" {
		req.Header.Set("Authorization", "Basic "+registryAuth)
	}
	authResponse, err := registryHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer authResponse.Body.Close()
	if authResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry token 服务返回 %s", authResponse.Status)
	}
	body, err := io.ReadAll(io.LimitReader(authResponse.Body, maxRegistryBodySize+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxRegistryBodySize {
		return "", errors.New("registry token 响应超过大小限制")
	}
	tokenResponse := &types.TokenResponse{}
	if err := json.Unmarshal(body, tokenResponse); err != nil {
		return "", err
	}
	token := tokenResponse.Token
	if token == "" {
		token = tokenResponse.AccessToken
	}
	if token == "" {
		return "", errors.New("registry token 响应中没有 token")
	}
	return "Bearer " + token, nil
}

func GetAuthURL(challenge string, imageRef ref.Named) (*url.URL, error) {
	trimmed := strings.TrimSpace(challenge)
	if scheme, rest, found := strings.Cut(trimmed, " "); found && strings.EqualFold(scheme, "bearer") {
		trimmed = rest
	}
	values, err := parseChallengeParameters(trimmed)
	if err != nil {
		return nil, err
	}
	realm := values["realm"]
	if realm == "" {
		return nil, errors.New("challenge header 缺少 realm")
	}
	authURL, err := url.Parse(realm)
	if err != nil || authURL.Scheme != "https" || authURL.Host == "" {
		return nil, errors.New("registry token realm 必须是有效的 HTTPS 地址")
	}
	query := authURL.Query()
	if service := values["service"]; service != "" {
		query.Set("service", service)
	}
	query.Set("scope", fmt.Sprintf("repository:%s:pull", ref.Path(imageRef)))
	authURL.RawQuery = query.Encode()
	return authURL, nil
}

func parseChallengeParameters(value string) (map[string]string, error) {
	parameters := make(map[string]string)
	for len(strings.TrimSpace(value)) > 0 {
		value = strings.TrimSpace(value)
		key, rest, found := strings.Cut(value, "=")
		if !found {
			return nil, errors.New("challenge 参数格式错误")
		}
		key = strings.ToLower(strings.TrimSpace(key))
		rest = strings.TrimSpace(rest)
		if key == "" || !strings.HasPrefix(rest, `"`) {
			return nil, errors.New("challenge 参数格式错误")
		}
		rest = rest[1:]
		end := -1
		escaped := false
		for index, character := range rest {
			if character == '\\' && !escaped {
				escaped = true
				continue
			}
			if character == '"' && !escaped {
				end = index
				break
			}
			escaped = false
		}
		if end < 0 {
			return nil, errors.New("challenge 参数引号未闭合")
		}
		parameters[key] = rest[:end]
		value = strings.TrimSpace(rest[end+1:])
		if value == "" {
			break
		}
		if !strings.HasPrefix(value, ",") {
			return nil, errors.New("challenge 参数缺少逗号")
		}
		value = value[1:]
	}
	return parameters, nil
}

func GetChallengeURL(imageRef ref.Named) (url.URL, error) {
	host, err := GetRegistryAddress(imageRef.Name())
	if err != nil {
		return url.URL{}, err
	}
	return url.URL{Scheme: "https", Host: host, Path: "/v2/"}, nil
}

func GetRegistryAddress(imageRef string) (string, error) {
	normalizedRef, err := ref.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", err
	}
	address := ref.Domain(normalizedRef)
	if address == DefaultRegistryDomain {
		address = DefaultRegistryHost
	}
	return address, nil
}
