package settings

import (
	"encoding/json"
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/logic/settings"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateAppSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body types.UpdateAppSettingsPartial
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
				"code": 400, "msg": "请求体无效", "data": map[string]interface{}{},
			})
			return
		}
		l := settings.NewUpdateAppSettingsLogic(r.Context(), svcCtx)
		resp, err := l.UpdateAppSettings(&body)
		writeLogicResp(w, r, resp, err)
	}
}
