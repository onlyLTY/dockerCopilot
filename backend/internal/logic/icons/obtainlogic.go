package icons

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ObtainLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewObtainLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ObtainLogic {
	return &ObtainLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ObtainLogic) Obtain() (*types.Resp, error) {
	icons, err := readIcons(iconConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			icons = map[string]string{}
		} else {
			l.Errorf("读取图标配置失败: %v", err)
			return nil, fmt.Errorf("failed to read config: %v", err)
		}
	}
	canonical := make(map[string]string, len(icons))
	keys := make([]string, 0, len(icons))
	for key := range icons {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := icons[key]
		repository, err := NormalizeRepository(key)
		if err != nil {
			continue
		}
		if current, exists := canonical[repository]; !exists || key == repository {
			canonical[repository] = value
		} else {
			_ = current
		}
	}
	return &types.Resp{Code: 200, Msg: "Success", Data: canonical}, nil
}
