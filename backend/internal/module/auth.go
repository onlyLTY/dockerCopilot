package module

import (
	"encoding/json"
	"errors"
	"fmt"
	ref "github.com/distribution/reference"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const ChallengeHeader = "WWW-Authenticate"
const (
	DefaultRegistryDomain = "docker.io"
	DefaultRegistryHost   = "index.docker.io"
)

var DefaultAcceleratorHostList = []string{"docker.1ms.run", "docker.m.daocloud.io",
	"docker.1panel.top", "docker.1panel.live", "proxy.1panel.live", "dockerproxy.1panel.live", "docker.1panel.dev",
	"docker.anye.in", "hub.rat.dev", "docker.amingg.com"}

func GetToken(image types.Image, registryAuth string) (string, error) {
	logx.Infof("image name %s", image.ImageName)
	normalizedRef, err := ref.ParseNormalizedNamed(image.ImageName)
	if err != nil {
		return "", err
	}

	URL := GetChallengeURL(normalizedRef)

	var req *http.Request
	if req, err = GetChallengeRequest(URL); err != nil {
		return "", err
	}

	client := &http.Client{}
	var res *http.Response
	if res, err = client.Do(req); err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logx.Error("GetToken关闭Body失败" + err.Error())
		}
	}(res.Body)
	v := res.Header.Get(ChallengeHeader)

	challenge := strings.ToLower(v)
	if strings.HasPrefix(challenge, "basic") {
		if registryAuth == "" {
			return "", fmt.Errorf("no credentials available")
		}

		return fmt.Sprintf("Basic %s", registryAuth), nil
	}
	if strings.HasPrefix(challenge, "bearer") {
		return GetBearerHeader(challenge, normalizedRef, registryAuth)
	}

	return "", errors.New("unsupported challenge type from registry")
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
		return "", err
	}

	if registryAuth != "" {
		logx.Info("私有镜像，无法获取是否有更新")
		r.Header.Add("Authorization", fmt.Sprintf("Basic %s", registryAuth))
	} else {
		logx.Info("No credentials found.")
	}

	var authResponse *http.Response
	if authResponse, err = client.Do(r); err != nil {
		return "", err
	}

	body, _ := io.ReadAll(authResponse.Body)
	tokenResponse := &types.TokenResponse{}

	err = json.Unmarshal(body, tokenResponse)
	if err != nil {
		return "", err
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

		return nil, fmt.Errorf("challenge header did not include all values needed to construct an auth url")
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
		if checkHost(DefaultRegistryHost) {
			address = DefaultRegistryHost
		} else {
			for _, host := range DefaultAcceleratorHostList {
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

func checkHost(host string) bool {
	URL := "https://" + host + "/v2/"
	// 创建带有超时设置的 http.Client
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	// 发送 HEAD 请求
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

	// 检查 HTTP 响应状态码
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusUnauthorized {
		return true
	}

	logx.Errorf("Failed to connect to %s: %s", URL, resp.Status)
	return false
}
