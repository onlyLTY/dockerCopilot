package icons

import (
	"errors"
	"net/http"
	"os"
	"regexp"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ObtainHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsPath := imageLogosPath
		logx.Infof("Reading icons from: %s", jsPath)

		imageLogosMu.Lock()
		contentBytes, err := readImageLogosConfig(jsPath)
		imageLogosMu.Unlock()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logx.Info("Config file does not exist, returning empty.")
				httpx.OkJsonCtx(r.Context(), w, types.Resp{
					Code: 200,
					Msg:  "Success",
					Data: "{}",
				})
				return
			}
			logx.Errorf("Error reading config: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, types.Resp{
				Code: http.StatusInternalServerError, Msg: "读取图标配置失败", Data: map[string]interface{}{},
			})
			return
		}

		content := string(contentBytes)
		// fmt.Printf("Config content: %s\n", content) // Uncomment for deep debug

		// 改进的正则表达式：匹配 "key": "value"，允许一定的格式变化
		// 使用反引号表示原始字符串。
		re := regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]+)"`)
		matches := re.FindAllStringSubmatch(content, -1)

		icons := make(map[string]string)
		for _, match := range matches {
			if len(match) == 3 {
				key := match[1]
				val := match[2]
				icons[key] = val
				// fmt.Printf("Found icon: %s -> %s\n", key, val)
			}
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
