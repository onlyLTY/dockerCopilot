package version

import (
	"context"
	"errors"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"time"
)

type UpdateProgramLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProgramLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProgramLogic {
	return &UpdateProgramLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProgramLogic) UpdateProgram() (resp *types.Resp, err error) {
	resp = &types.Resp{}
	err = utiles.UpdateProgram(l.ctx)
	if err != nil {
		if errors.Is(err, utiles.ErrAlreadyLatest) || errors.Is(err, utiles.ErrImageManagedUpdate) || errors.Is(err, utiles.ErrRemoteVersionNotNewer) {
			resp.Code = 409
			resp.Msg = err.Error()
			resp.Data = map[string]interface{}{}
			return resp, err
		}
		resp.Code = 500
		resp.Msg = "程序更新失败，请查看服务日志"
		resp.Data = map[string]interface{}{}
		l.Errorf("程序更新失败: %v", err)
		return resp, err
	}
	resp.Code = 200
	resp.Msg = "success"
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	resp.Data = map[string]interface{}{}
	return resp, nil
}
