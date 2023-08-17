package utiles

import (
	"fmt"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	versionURL     = "https://ghproxy.com/https://raw.githubusercontent.com/onlyLTY/oneKeyUpdate/UGREEN/version"
	releaseBaseURL = "https://ghproxy.com/https://github.com/onlyLTY/oneKeyUpdate/releases/download"
)

func UpdateProgram(ctx *svc.ServiceContext) (types.MsgResp, error) {
	resp, err := http.Get(versionURL)
	if err != nil {
		logx.Info("Error fetching version info:", err)
		return types.MsgResp{}, nil
	}
	defer resp.Body.Close()

	versionData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logx.Info("Error reading version data:", err)
		return types.MsgResp{}, nil
	}

	version := strings.TrimSpace(string(versionData))

	// 2. 构造下载链接
	downloadURL := fmt.Sprintf("%s/%s/onekeyupdate", releaseBaseURL, version)

	// 3. 下载文件
	resp, err = http.Get(downloadURL)
	if err != nil {
		logx.Info("Error downloading the asset:", err)
		return types.MsgResp{}, nil
	}
	defer resp.Body.Close()

	outFile, err := os.Create("onekeyupdate")
	if err != nil {
		logx.Info("Error creating the asset file:", err)
		return types.MsgResp{}, nil
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		logx.Info("Error saving the asset:", err)
		return types.MsgResp{}, nil
	}

	logx.Info("Downloaded successfully!")
	return types.MsgResp{}, nil
}
