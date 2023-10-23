package utiles

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"io/ioutil"
	"net/http"
)

//var domain = "http://127.0.0.1:9123"

var domain = "http://127.0.0.1:10000"

type Request struct {
	Node   string `path:"node"`
	Body   string `json:"body"` // json 请求体，这里以 string 为例，你可以使用任意类型
	Header string `header:"X-Header"`
}

func GetNewJwt(ctx *svc.ServiceContext) (jwt, endpointid string, err error) {
	var EndpointsIDresponse []map[string]interface{}
	if ctx.PortainerJwt != "" {
		resp, err := GetEndpointsID(ctx)

		if err == nil && resp.StatusCode == http.StatusOK {

			err = json.NewDecoder(resp.Body).Decode(&EndpointsIDresponse)
			if err != nil {
				return "", "", fmt.Errorf("解析 JSON 错误: %v", err)
			}

			if len(EndpointsIDresponse) == 0 {
				return "", "", fmt.Errorf("未找到 endpoints 信息")
			}

			// 获取第一个 endpoints 的 ID
			endpointid := fmt.Sprintf("%v", EndpointsIDresponse[0]["Id"])
			return ctx.PortainerJwt, endpointid, nil
		}
		defer resp.Body.Close()
	}
	logx.Info("未找到jwt或jwt已失效，重新获取jwt")
	hash := md5.New()
	hash.Write([]byte(ctx.Config.Account))
	md5Str := hex.EncodeToString(hash.Sum(nil))
	// 创建 HTTP 客户端
	client := &http.Client{}

	// 创建 JSON 请求体
	requestBody := map[string]string{
		"username": "admin",
		"password": md5Str,
	}

	// 将请求体转换为 JSON 格式
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", err
	}

	// 创建 HTTP POST 请求
	req, err := http.NewRequest("POST", domain+"/api/auth", nil)
	if err != nil {
		return "", "", err
	}

	// 设置请求头为 JSON 格式
	req.Header.Set("Content-Type", "application/json")

	// 设置请求体
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	// 判断响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New(fmt.Sprintf("请求错误，状态码:%v", resp.StatusCode))
	}

	// 创建 map 解析响应体的 JSON
	var response map[string]string
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return "", "", err
	}
	ctx.PortainerJwt = response["jwt"]
	resp, err = GetEndpointsID(ctx)
	if err != nil {
		return "", "", err
	}
	err = json.NewDecoder(resp.Body).Decode(&EndpointsIDresponse)
	if err != nil {
		return "", "", fmt.Errorf("解析 JSON 错误: %v", err)
	}

	if len(EndpointsIDresponse) == 0 {
		return "", "", fmt.Errorf("未找到 endpoints 信息")
	}

	// 获取第一个 endpoints 的 ID
	endpointid = fmt.Sprintf("%v", EndpointsIDresponse[0]["Id"])
	return response["jwt"], endpointid, nil
}

func GetEndpointsID(svc *svc.ServiceContext) (*http.Response, error) {
	// 从请求的 session 中获取 jwt
	jwt := svc.PortainerJwt

	// 创建 HTTP 客户端
	client := &http.Client{}

	// 创建 HTTP GET 请求
	endpointsURL := domain + "/api/endpoints"
	req, err := http.NewRequest("GET", endpointsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求错误: %v", err)
	}

	// 设置请求头的 Authorization 字段为 jwt
	req.Header.Set("Authorization", jwt)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求发送错误: %v", err)
	}
	return resp, nil
}
