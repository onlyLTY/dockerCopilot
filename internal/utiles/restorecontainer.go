package utiles

import (
	"bytes"
	"context"
	"encoding/json"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"log"
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	for i, containerInfo := range configList {
		info := "正在恢复第" + strconv.Itoa(i+1) + "个容器"
		ctx.ProgressStore[taskID] = svc.TaskProgress{
			Percentage: 0,
			Message:    info,
		}
		cli.NegotiateAPIVersion(context.TODO())
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
			log.Fatal(err)
		}
		baseURL := domain + "/api/endpoints/" + endpointsId
		createURL := baseURL + "/docker/containers/create?name=" + containerInfo.Name
		createReq, err := http.NewRequestWithContext(context.Background(), "POST", createURL, bytes.NewBuffer(postData))
		if err != nil {
			log.Fatal(err)
		}
		createReq.Header.Set("Authorization", "Bearer "+jwt)
		createReq.Header.Set("Content-Type", "application/json")
		createResp, err := http.DefaultClient.Do(createReq)
		if err != nil {
			log.Fatal(err)
		}
		defer createResp.Body.Close()

		_, err = io.ReadAll(createResp.Body)
		if err != nil {
			logx.Error("Failed to create container: %s", err)
			info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
			ctx.ProgressStore[taskID] = svc.TaskProgress{
				Percentage: 0,
				Message:    info,
			}
			backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
		} else {
			switch createResp.StatusCode {
			case http.StatusOK:
				backupList = append(backupList, containerInfo.Name+"恢复成功")
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
			case http.StatusBadRequest:
				logx.Error("Failed to create container: %s", err)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			case http.StatusNotFound:
				logx.Error("Failed to create container: %s", err)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			case http.StatusConflict:
				logx.Error("Failed to create container: %s", err)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			case http.StatusInternalServerError:
				logx.Error("Failed to create container: %s", err)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			default:
				logx.Error("Failed to create container: %s", err)
				info = "正在恢复第" + strconv.Itoa(i+1) + "个容器"
				ctx.ProgressStore[taskID] = svc.TaskProgress{
					Percentage: 0,
					Message:    info,
				}
				backupList = append(backupList, containerInfo.Name+"恢复失败"+err.Error())
			}

		}

	}
	ctx.ProgressStore[taskID] = svc.TaskProgress{
		Percentage: 100,
		Message:    strings.Join(backupList, "\n"),
	}
	return nil
}
