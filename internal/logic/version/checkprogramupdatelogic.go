package version

import (
	"context"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/config"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckprogramupdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckprogramupdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckprogramupdateLogic {
	return &CheckprogramupdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

const versionURL = "https://ghproxy.com/https://raw.githubusercontent.com/onlyLTY/oneKeyUpdate/UGREEN/version"

func (l *CheckprogramupdateLogic) CheckProgramUpdate() (resp *types.MsgResp, err error) {
	remoteVersion, err := fetchVersionFromURL(versionURL)
	if err != nil {
		logx.Info("获取版本错误", err)
		return
	}

	localVersion := config.Version

	if localVersion == remoteVersion {
		logx.Info("版本一致:", localVersion)
		return &types.MsgResp{Msg: "not need"}, nil
	} else {
		logx.Infof("版本不一致! 本地: %s, 远程: %s\n", localVersion, remoteVersion)
		return &types.MsgResp{Msg: "need"}, nil
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
