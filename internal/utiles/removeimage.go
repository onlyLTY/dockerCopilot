package utiles

import (
	"encoding/json"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"net/http"
	"strconv"
)

func RemoveImage(ctx *svc.ServiceContext, imageNameAndTag string, force bool) (types.MsgResp, error) {
	jwt, endpointsId, err := GetNewJwt(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/images/"
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return types.MsgResp{}, err
	}
	req.Header.Add("Authorization", jwt)
	params := map[string]bool{
		"force": force,
	}
	query := req.URL.Query()
	for k, v := range params {
		query.Add(k, strconv.FormatBool(v))
	}
	req.URL.RawQuery = query.Encode()
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
