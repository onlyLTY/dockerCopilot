package container

import (
	"errors"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

var errContainerOperationInProgress = errors.New("该容器正在执行其他操作，或系统正在恢复备份，请稍后重试")

func beginContainerOperation(serviceContext *svc.ServiceContext, resp *types.Resp, rawID, operation string) (string, error) {
	containerID := strings.TrimSpace(rawID)
	if containerID == "" {
		resp.Code = 400
		resp.Msg = "容器 ID 不能为空"
		resp.Data = map[string]interface{}{}
		return "", errors.New(resp.Msg)
	}
	if !serviceContext.BeginContainerOperation(containerID, operation) {
		resp.Code = 409
		resp.Msg = errContainerOperationInProgress.Error()
		resp.Data = map[string]interface{}{}
		return "", errContainerOperationInProgress
	}
	return containerID, nil
}
