package settings

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProxySettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProxySettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProxySettingsLogic {
	return &UpdateProxySettingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProxySettingsLogic) UpdateProxySettings(req *types.ProxySettingsData) (*types.Resp, error) {
	updated, err := settingstore.SetProxySettings(settingstore.ProxySettings{
		GithubProxy: req.GithubProxy,
		HTTPProxy:   req.HTTPProxy,
		HTTPSProxy:  req.HTTPSProxy,
		NoProxy:     req.NoProxy,
	})
	if err != nil {
		return logic.Biz(400, err.Error(), map[string]interface{}{}), nil
	}
	return logic.Biz(200, "代理设置已保存，重启服务后生效", updated), nil
}
