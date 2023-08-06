package utiles

import (
	"encoding/json"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"net/http"
	"strconv"
)

func StartContainer(ctx *svc.ServiceContext, name string) (types.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	jwt, endpointsId, err := GetNewJwt(ctx)
	containerID, err := findContainerIDByName(containers, name)
	if err != nil {
		return types.MsgResp{}, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/containers/" + containerID + "/start"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return types.MsgResp{}, err
	}
	req.Header.Add("Authorization", jwt)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return types.MsgResp{}, nil
	}
	defer response.Body.Close()
	resp := types.MsgResp{Status: strconv.Itoa(response.StatusCode), Msg: response.Status}
	type ErrorResponse struct {
		Message string `json:"message"`
	}
	// 对于204和304，我们不需要尝试解析响应体中的内容
	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusNotModified {
		// 对于其他状态码，我们尝试解析响应体中的JSON错误消息
		errorResponse := ErrorResponse{}
		err = json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			// 在此处处理JSON解码错误
			return types.MsgResp{}, err
		}
		// 如果解析成功，将错误消息设置为resp的Msg字段
		resp.Msg = errorResponse.Message
	}
	return resp, nil
}