package utiles

import (
	"bytes"
	"context"
	"encoding/json"
	docker "github.com/docker/docker/api/types"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RestoreContainer(ctx *svc.ServiceContext, filename string, taskID string) error {
	var backupList []string
	basePath := `D:\MyProject\oneKeyUpdateGo`
	fullPath := filepath.Join(basePath, filename+".json")

	content, err := os.ReadFile(fullPath)
	if err != nil {
		logx.Error("Failed to read file: %s", err)
		return err
	}
	var configList []docker.ContainerCreateConfig
	err = json.Unmarshal(content, &configList)
	if err != nil {
		logx.Error("Failed to parse json: %s", err)
		return err
	}
	for i, containerInfo := range configList {
		info := "正在恢复第" + strconv.Itoa(i+1) + "个容器"
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    info,
		}
		if err != nil {
			backupList = append(backupList, "出现错误"+err.Error())
			return err
		}
		jwt, endpointsId, err := GetNewJwt(ctx)
		if err != nil {
			return err
		}
		url := domain + "/api/endpoints/" + endpointsId + "/docker/images/create"
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", jwt)
		params := map[string]string{
			"fromImage": containerInfo.Config.Image,
		}
		query := req.URL.Query()
		for k, v := range params {
			query.Add(k, v)
		}
		req.URL.RawQuery = query.Encode()
		client := &http.Client{}
		response, err := client.Do(req)
		if err != nil {
			backupList = append(backupList, containerInfo.Config.Image+"拉取镜像出现错误"+err.Error())
			logx.Errorf("Failed to pull image: %s", err)
		}
		defer response.Body.Close()
		decodePullResp(response.Body, ctx, taskID)
		body := configWrapper{
			Config:           containerInfo.Config,
			HostConfig:       containerInfo.HostConfig,
			NetworkingConfig: containerInfo.NetworkingConfig,
		}
		postData, err := json.Marshal(body)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
		}
		baseURL := domain + "/api/endpoints/" + endpointsId
		createURL := baseURL + "/docker/containers/create?name=" + containerInfo.Name
		createReq, err := http.NewRequestWithContext(context.Background(), "POST", createURL, bytes.NewBuffer(postData))
		if err != nil {
			logx.Error("Failed to create container: %s", err)
		}
		createReq.Header.Set("Authorization", "Bearer "+jwt)
		createReq.Header.Set("Content-Type", "application/json")
		createResp, err := http.DefaultClient.Do(createReq)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logx.Error("defer error: %s", err)
			}
		}(createResp.Body)

		createData, err := io.ReadAll(createResp.Body)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 0,
				Message:    info,
			}
			backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
		} else {
			switch createResp.StatusCode {
			case http.StatusOK:
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复成功")
			case http.StatusBadRequest:
				logx.Error("Failed to create container: %s", string(createData))
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
			case http.StatusNotFound:
				logx.Error("Failed to create container: %v", createResp)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
			case http.StatusConflict:
				logx.Error("Failed to create container: %v", createResp)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
			case http.StatusInternalServerError:
				logx.Error("Failed to create container: %v", createResp)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
			default:
				logx.Error("Failed to create container: %v", createResp)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+string(createData))
			}

		}

	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 100,
		Message:    strings.Join(backupList, "\n"),
	}
	return nil
}
