package utiles

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	myTypes "github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type configWrapper struct {
	*container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

func CreateContainer(ctx *svc.ServiceContext, oldName string, newName string, imageNameAndTag string) (myTypes.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	jwtToken, endpointsId, err := GetNewJwt(ctx)
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + jwtToken
	containerID, err := findContainerIDByName(containers, oldName)
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

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("读取响应体失败")
		log.Fatal(err)
	}

	var inspectedContainer types.ContainerJSON
	err = json.Unmarshal(data, &inspectedContainer)
	if err != nil {
		log.Println("解析响应体失败")
		log.Fatal(err)
	}

	inspectedContainer.Config.Hostname = ""
	inspectedContainer.Config.Image = imageNameAndTag
	inspectedContainer.Image = imageNameAndTag
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: inspectedContainer.NetworkSettings.Networks,
	}
	body := configWrapper{
		Config:           inspectedContainer.Config,
		HostConfig:       inspectedContainer.HostConfig,
		NetworkingConfig: networkingConfig,
	}
	postData, err := json.Marshal(body)
	if err != nil {
		log.Fatal(err)
	}
	createURL := baseURL + "/docker/containers/create?name=" + newName
	createReq, err := http.NewRequestWithContext(context.Background(), "POST", createURL, bytes.NewBuffer(postData))
	if err != nil {
		log.Fatal(err)
	}
	createReq.Header.Set("Authorization", "Bearer "+jwtToken)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		log.Fatal(err)
	}
	defer createResp.Body.Close()

	createData, err := ioutil.ReadAll(createResp.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Response from create:", string(createData))
	var responseMsg myTypes.MsgResp
	switch createResp.StatusCode {
	case http.StatusOK:
		responseMsg = myTypes.MsgResp{Status: "200", Msg: "容器创建成功"}
	case http.StatusBadRequest:
		responseMsg = myTypes.MsgResp{Status: "400", Msg: "请求错误请重试"}
	case http.StatusNotFound:
		responseMsg = myTypes.MsgResp{Status: "404", Msg: "没有找到这个镜像"}
	case http.StatusConflict:
		responseMsg = myTypes.MsgResp{Status: "409", Msg: "存在冲突"}
	case http.StatusInternalServerError:
		responseMsg = myTypes.MsgResp{Status: "500", Msg: "docker服务异常"}
	default:
		responseMsg = myTypes.MsgResp{Status: strconv.Itoa(createResp.StatusCode), Msg: "未知错误"}
	}
	return responseMsg, nil
}
