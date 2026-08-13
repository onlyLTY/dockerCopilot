package settings

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/daemonhelper"
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type DaemonProxyRequest struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NoProxy    string `json:"noProxy"`
	Hash       string `json:"hash"`
}

type DaemonProxyData struct {
	HelperEnabled   bool                     `json:"helperEnabled"`
	FileExists      bool                     `json:"fileExists"`
	Writable        bool                     `json:"writable"`
	Hash            string                   `json:"hash"`
	FileProxy       daemonhelper.ProxyValues `json:"fileProxy"`
	EffectiveProxy  daemonhelper.ProxyValues `json:"effectiveProxy"`
	RestartRequired bool                     `json:"restartRequired"`
	Message         string                   `json:"message,omitempty"`
}

type DaemonProxyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	helper *daemonhelper.Client
}

func NewDaemonProxyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DaemonProxyLogic {
	return &DaemonProxyLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, helper: daemonhelper.New()}
}

func (l *DaemonProxyLogic) Get() (*types.Resp, error) {
	if err := l.svcCtx.RequireDocker(); err != nil {
		return logic.Biz(errorx.CodeDockerUnavailable, "Docker 服务不可用", map[string]interface{}{}), err
	}
	info, err := l.svcCtx.DockerClient.Info(l.ctx)
	if err != nil {
		logx.Errorf("read daemon proxy info failed: %v", err)
		return logic.Biz(errorx.CodeDockerUnavailable, "读取 Docker daemon 代理状态失败", map[string]interface{}{}), nil
	}
	status, helperErr := l.helper.Status(l.ctx)
	if helperErr != nil {
		return logic.Biz(200, "Docker daemon 当前代理已读取，但宿主机 helper 未启用", DaemonProxyData{
			EffectiveProxy: daemonhelper.ProxyValues{HTTPProxy: info.HTTPProxy, HTTPSProxy: info.HTTPSProxy, NoProxy: info.NoProxy},
			Message:        "宿主机 helper 未启用，无法读取或覆写 daemon.json",
		}), nil
	}
	return logic.Biz(200, "success", daemonProxyData(status, daemonhelper.ProxyValues{
		HTTPProxy: info.HTTPProxy, HTTPSProxy: info.HTTPSProxy, NoProxy: info.NoProxy,
	})), nil
}

func (l *DaemonProxyLogic) Apply(req *DaemonProxyRequest) (*types.Resp, error) {
	if _, err := l.helper.Status(l.ctx); err != nil {
		return logic.Biz(errorx.CodeNotFound, "宿主机代理 helper 未启用或不可用", map[string]interface{}{}), nil
	}
	result, err := l.helper.Apply(l.ctx, daemonhelper.ApplyRequest{
		ProxyValues: daemonhelper.ProxyValues{HTTPProxy: req.HTTPProxy, HTTPSProxy: req.HTTPSProxy, NoProxy: req.NoProxy},
		Hash:        req.Hash,
	})
	if err != nil {
		return helperError(err)
	}
	if _, err := settingstore.SetDaemonProxyDraft(settingstore.DaemonProxyDraft{
		HTTPProxy: req.HTTPProxy, HTTPSProxy: req.HTTPSProxy, NoProxy: req.NoProxy,
	}); err != nil {
		logx.Errorf("save daemon proxy draft after apply failed: %v", err)
	}
	return logic.Biz(200, "Docker daemon 代理已写入 daemon.json，重启后才会生效", result), nil
}

func (l *DaemonProxyLogic) Restart() (*types.Resp, error) {
	result, err := l.helper.Restart(l.ctx)
	if err != nil {
		return helperError(err)
	}
	return logic.Biz(202, result.Message, result), nil
}

func (l *DaemonProxyLogic) Operation(operationID string) (*types.Resp, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" || strings.ContainsAny(operationID, "/\\") {
		return logic.Biz(errorx.CodeBadRequest, "重启任务编号无效", map[string]interface{}{}), nil
	}
	result, err := l.helper.Operation(l.ctx, operationID)
	if err != nil {
		return helperError(err)
	}
	return logic.Biz(200, result.Message, result), nil
}

func daemonProxyData(status daemonhelper.Status, effective daemonhelper.ProxyValues) DaemonProxyData {
	return DaemonProxyData{
		HelperEnabled: status.Enabled, FileExists: status.FileExists, Writable: status.Writable,
		Hash: status.Hash, FileProxy: status.FileProxy, EffectiveProxy: effective,
		RestartRequired: status.RestartRequired, Message: status.Message,
	}
}

func helperError(err error) (*types.Resp, error) {
	var httpErr *daemonhelper.HTTPError
	if errors.As(err, &httpErr) {
		code := errorx.CodeInternal
		switch httpErr.StatusCode {
		case http.StatusBadRequest:
			code = errorx.CodeBadRequest
		case http.StatusConflict:
			code = errorx.CodeConflict
		case http.StatusNotFound:
			code = errorx.CodeNotFound
		}
		return logic.Biz(code, httpErr.Message, map[string]interface{}{}), nil
	}
	return logic.Biz(errorx.CodeNotFound, "宿主机代理 helper 未启用或不可用", map[string]interface{}{}), nil
}
