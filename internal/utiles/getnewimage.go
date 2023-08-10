package utiles

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"io"
	"net/http"
	"strconv"
)

func GetNewImage(ctx *svc.ServiceContext, imageNameAndTag string) (types.MsgResp, error) {
	jwt, endpointsId, err := GetNewJwt(ctx)
	if err != nil {
		return types.MsgResp{}, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/images/create"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return types.MsgResp{}, err
	}
	req.Header.Add("Authorization", jwt)
	params := map[string]string{
		"fromImage": imageNameAndTag,
	}
	query := req.URL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return types.MsgResp{}, nil
	}
	defer response.Body.Close()
	// 这里是流传输，得想办法处理
	// 使用bufio读取响应体
	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			fmt.Println("读取错误:", err)
			break
		}
		if err == io.EOF {
			break
		}

		// 打印每行响应或进行其他处理
		fmt.Println(string(line))
	}

	// 打印响应状态
	fmt.Println("响应状态:", response.Status)
	resp := types.MsgResp{Status: strconv.Itoa(response.StatusCode), Msg: response.Status}
	type ErrorResponse struct {
		Message string `json:"message"`
	}
	// 对于204和304，我们不需要尝试解析响应体中的内容
	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated {
		// 处理成功的响应
		fmt.Println("拉取成功")
		resp := types.MsgResp{Status: strconv.Itoa(response.StatusCode), Msg: "成功"}
		return resp, nil
	} else if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusNotModified {
		// 对于错误的响应，尝试解析错误消息
		errorResponse := ErrorResponse{}
		err = json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			// 在此处处理JSON解码错误
			return types.MsgResp{}, err
		}
		// 如果解析成功，将错误消息设置为resp的Msg字段
		resp.Msg = errorResponse.Message
		return resp, nil
	} else {
		// 处理其他情况，例如无内容的响应或不修改的响应
		resp := types.MsgResp{Status: strconv.Itoa(response.StatusCode), Msg: response.Status}
		return resp, nil
	}

}
