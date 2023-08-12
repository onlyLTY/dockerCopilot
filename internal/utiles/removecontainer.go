package utiles

import (
	"encoding/json"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/types"
	"net/http"
	"strconv"
)

func RemoveContainer(ctx *svc.ServiceContext, name string) (types.MsgResp, error) {
	containers, err := GetContainerList(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	jwt, endpointsId, err := GetNewJwt(ctx)
	containerID, err := findContainerIDByName(containers, name)
	if err != nil {
		return types.MsgResp{}, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/containers/" + containerID + "?v=1"
	req, err := http.NewRequest("DELETE", url, nil)
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
	// 对于204不解析
	if response.StatusCode != http.StatusNoContent {
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
