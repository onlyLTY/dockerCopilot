package utiles

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/config"
)

var updateValuePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var updateRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

const defaultUpdateRepository = "autunn/dockerCopilot"

func GetRemoteVersion(ctx context.Context) (string, error) {
	channel, versionFile, err := updateChannel()
	if err != nil {
		return "", err
	}
	repository, err := updateRepository()
	if err != nil {
		return "", err
	}
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repository, channel, versionFile)
	versionURL, err := githubURL(baseURL)
	if err != nil {
		return "", err
	}
	client := newHTTPSClient(15 * time.Second)
	return fetchVersionFromURL(ctx, client, versionURL)
}

func updateRepository() (string, error) {
	repository := strings.TrimSpace(os.Getenv("UPDATE_REPOSITORY"))
	if repository == "" {
		repository = defaultUpdateRepository
	}
	if !updateRepositoryPattern.MatchString(repository) {
		return "", errors.New("UPDATE_REPOSITORY 必须使用 owner/repository 格式")
	}
	return repository, nil
}

func newHTTPSClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return errors.New("拒绝重定向到非 HTTPS 地址")
			}
			if len(via) >= 10 {
				return errors.New("HTTP 重定向次数过多")
			}
			return nil
		},
	}
}

func currentVersion() string {
	return strings.TrimSpace(config.Version)
}

func updateChannel() (channel, versionFile string, err error) {
	channel = strings.TrimSpace(os.Getenv("UPDATE_CHANNEL"))
	if channel == "" {
		switch {
		case strings.Contains(strings.ToUpper(currentVersion()), "UGREEN"):
			channel = "UGREEN"
		case strings.Contains(strings.ToUpper(currentVersion()), "FNOS"):
			channel = "FNOS"
		default:
			channel = "latest"
		}
	}
	if !updateValuePattern.MatchString(channel) {
		return "", "", errors.New("UPDATE_CHANNEL 格式错误")
	}
	switch strings.ToUpper(channel) {
	case "UGREEN":
		versionFile = "ugreen_version"
	case "FNOS":
		versionFile = "fn_version"
	default:
		versionFile = "version"
	}
	return channel, versionFile, nil
}

func githubURL(target string) (string, error) {
	proxy := strings.TrimSpace(os.Getenv("githubProxy"))
	if proxy != "" {
		target = strings.TrimRight(proxy, "/") + "/" + target
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("githubProxy 必须生成有效的 HTTPS 地址")
	}
	return parsed.String(), nil
}

func fetchVersionFromURL(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("版本服务器返回 %s", resp.Status)
	}
	versionData, err := io.ReadAll(io.LimitReader(resp.Body, 1025))
	if err != nil {
		return "", err
	}
	if len(versionData) > 1024 {
		return "", errors.New("版本响应超过大小限制")
	}
	version := strings.TrimSpace(string(versionData))
	if !updateValuePattern.MatchString(version) {
		return "", errors.New("远程版本格式错误")
	}
	return version, nil
}
