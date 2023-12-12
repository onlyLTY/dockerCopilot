package version

import (
	"context"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/config"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckForUpdatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckForUpdatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckForUpdatesLogic {
	return &CheckForUpdatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckForUpdatesLogic) CheckForUpdates() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	githubProxy := os.Getenv("githubProxy")
	if githubProxy != "" {
		githubProxy = strings.TrimRight(githubProxy, "/") + "/"
	}
	versionURL := githubProxy + "https://raw.githubusercontent.com/onlyLTY/dockerCopilot/UGREEN/version"
	remoteVersion, err := fetchVersionFromURL(versionURL)
	if err != nil {
		logx.Info("获取版本错误", err)
		resp.Code = 500
		resp.Msg = "获取版本错误" + err.Error()
		return resp, err
	}

	localVersion := config.Version

	if localVersion == remoteVersion {
		logx.Info("版本一致:", localVersion)
		resp.Code = 200
		resp.Msg = "not need"
		return resp, nil
	} else {
		logx.Infof("版本不一致! 本地: %s, 远程: %s\n", localVersion, remoteVersion)
		resp.Code = 200
		resp.Msg = "need"
		return resp, nil
	}

}

func fetchVersionFromURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	versionData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(versionData)), nil
}
