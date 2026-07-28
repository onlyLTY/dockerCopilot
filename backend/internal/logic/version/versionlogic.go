package version

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/config"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
)

type VersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VersionLogic {
	return &VersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VersionLogic) Version(req *types.VersionReq) (resp *types.Resp, err error) {
	resp = &types.Resp{}
	if req.Type == "local" {
		resp.Code = 200
		resp.Msg = "success"
		resp.Data = map[string]string{
			"version":   config.Version,
			"buildDate": config.BuildDate,
		}
		return resp, nil
	} else if req.Type == "remote" {
		remoteVersion, err := utiles.GetRemoteVersion()
		if err != nil {
			resp.Code = 50001
			resp.Msg = "获取版本错误" + err.Error()
			resp.Data = map[string]string{
				"remoteVersion": config.Version,
			}
			return resp, nil
		} else if remoteVersion != config.Version {
			resp.Code = 200
			resp.Msg = "程序有更新"
			resp.Data = map[string]string{
				"remoteVersion": remoteVersion,
			}
			return resp, nil
		} else {
			resp.Code = 200
			resp.Msg = "程序无更新"
			resp.Data = map[string]string{
				"remoteVersion": remoteVersion,
			}
			return resp, nil
		}

	} else {
		resp.Code = 400
		resp.Msg = "type 参数错误"
		resp.Data = map[string]string{}
		return resp, nil
	}
}
