package handler

import (
	"encoding/json"
	"net/http"

	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetProxySettingsHandler(w http.ResponseWriter, r *http.Request) {
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": settingstore.GetProxySettings(),
	})
}

func UpdateProxySettingsHandler(w http.ResponseWriter, r *http.Request) {
	var settings settingstore.ProxySettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	updated, err := settingstore.SetProxySettings(settings)
	if err != nil {
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
			"data": map[string]interface{}{},
		})
		return
	}
	httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
		"code": 200,
		"msg":  "代理设置已保存，重启服务后生效",
		"data": updated,
	})
}
