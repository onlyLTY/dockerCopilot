package utiles

import (
	"context"
	"encoding/json"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func BackupContainer(ctx *svc.ServiceContext) ([]string, error) {
	var errList []string
	jwtToken, endpointsId, err := GetNewJwt(ctx)
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + jwtToken
	containerList, err := GetContainerList(ctx)
	if err != nil {
		return nil, err
	}
	var backerupList []docker.ContainerCreateConfig
	for i, v := range containerList {
		containerID := containerList[i].ID
		baseURL := domain + "/api/endpoints/" + endpointsId
		url := baseURL + "/docker/containers/" + containerID + "/json"
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			log.Println("创建请求失败")
			log.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("获取容器信息失败")
			log.Fatal(err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("读取响应体失败")
			log.Fatal(err)
		}

		var inspectedContainer docker.ContainerJSON
		err = json.Unmarshal(data, &inspectedContainer)

		if err != nil {
			log.Println("获取容器信息失败")
			log.Fatal(err)
		}
		var containerName, imageNameAndTag string
		if len(v.Names) > 0 {
			containerName = v.Names[0][1:]
		} else {
			containerName = "get container name error"
			logx.Error("get container name error" + v.ID)
		}
		if v.Image != "" {
			imageNameAndTag = v.Image
		} else {
			imageNameAndTag = v.ImageID
			errList = append(errList, containerName+"镜像格式错误，请手动修正")
			logx.Error("image dont have name" + v.ID)
		}
		inspectedContainer.Config.Hostname = ""
		inspectedContainer.Config.Image = imageNameAndTag
		inspectedContainer.Image = imageNameAndTag
		config := inspectedContainer.Config
		hostConfig := inspectedContainer.HostConfig
		networkingConfig := &network.NetworkingConfig{
			EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
		}
		createConfig := docker.ContainerCreateConfig{Config: config, HostConfig: hostConfig, NetworkingConfig: networkingConfig, Name: containerName}
		backerupList = append(backerupList, createConfig)
	}
	jsonData, err := json.MarshalIndent(backerupList, "", "  ")
	if err != nil {
		logx.Error("Error marshalling data:", err)
		return nil, err
	}
	backupDir := `/data/backups`
	currentDate := time.Now().Format("2006-01-02")
	fileName := "backup-" + currentDate + ".json"
	fullPath := filepath.Join(backupDir, fileName)
	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		logx.Error("Error writing to file:", err)
		return nil, err
	}
	return errList, nil
}
