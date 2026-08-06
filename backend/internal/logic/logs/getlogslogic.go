package logs

import (
	"context"

	"github.com/onlyLTY/dockerCopilot/internal/logic"
	"github.com/onlyLTY/dockerCopilot/internal/logstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLogsLogic {
	return &GetLogsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetLogsLogic) GetLogs(req *types.GetLogsReq) (*types.Resp, error) {
	entries, err := logstore.ReadRecent(req.Limit, req.Level)
	if err != nil {
		return logic.Fail(&types.Resp{}, err, 500, "读取日志失败")
	}
	return logic.Biz(200, "success", map[string]interface{}{
		"entries": entries,
		"level":   req.Level,
		"limit":   req.Limit,
	}), nil
}
