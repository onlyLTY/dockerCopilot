package icons

import (
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ObtainHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		icons, err := obtainImageLogos()
		if err != nil {
			logx.Errorf("Error reading config: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, types.Resp{
				Code: http.StatusInternalServerError, Msg: "读取图标配置失败", Data: map[string]interface{}{},
			})
			return
		}

		logx.Infof("Total icons found: %d", len(icons))

		response := struct {
			Code int               `json:"code"`
			Msg  string            `json:"msg"`
			Data map[string]string `json:"data"`
		}{
			Code: 200,
			Msg:  "Success",
			Data: icons,
		}

		httpx.OkJsonCtx(r.Context(), w, response)
	}

}
