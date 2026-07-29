package icons

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ObtainHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icons, err := readIcons(iconConfigPath())
		if err != nil {
			if os.IsNotExist(err) {
				icons = map[string]string{}
			} else {
				logx.Errorf("读取图标配置失败: %v", err)
				httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to read config: %v", err))
				return
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
			repository, err := normalizeRepository(key)
			if err != nil {
				continue
			}
			if current, exists := canonical[repository]; !exists || key == repository {
				canonical[repository] = value
			} else {
				_ = current
			}
		}
		httpx.OkJsonCtx(r.Context(), w, types.Resp{Code: 200, Msg: "Success", Data: canonical})
	}
}
