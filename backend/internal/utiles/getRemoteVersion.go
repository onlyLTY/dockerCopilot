package utiles

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

func GetRemoteVersion() (remoteVersion string, err error) {
	versionURL, err := OfficialVersionURL()
	if err != nil {
		return "0.0.0", err
	}
	remoteVersion, err = fetchVersionFromURL(versionURL)
	if err != nil {
		return "0.0.0", err
	}

	localVersion := config.Version
	if strings.Contains(localVersion, "FNOS") {
		logx.Infof("飞牛版本，无需在线更新")
		return localVersion, nil
	}
	if localVersion == remoteVersion {
		logx.Info("版本一致:", localVersion)
		return remoteVersion, nil
	}
	logx.Infof("版本不一致! 本地: %s, 远程: %s\n", localVersion, remoteVersion)
	return remoteVersion, nil
}

func fetchVersionFromURL(rawURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			logx.Error("关闭Body失败:", err)
		}
	}(resp.Body)

	versionData, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(versionData)), nil
}
