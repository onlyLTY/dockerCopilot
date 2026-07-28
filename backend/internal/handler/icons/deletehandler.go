package icons

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("imageName")
		if key == "" {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("imageName is required"))
			return
		}
		if strings.ContainsAny(key, "\\/\"'") {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("invalid imageName"))
			return
		}
		jsPath := "/data/icon/imageLogos.js"
		content, err := os.ReadFile(jsPath)
		if err != nil && !os.IsNotExist(err) {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		text := string(content)
		re := regexp.MustCompile(`(?m)^\s*"` + regexp.QuoteMeta(key) + `"\s*:\s*"([^"]+)"\s*,?\s*\r?\n?`)
		match := re.FindStringSubmatch(text)
		if len(match) == 2 {
			_ = os.Remove(filepath.Join("/data/icon/icons", filepath.Base(match[1])))
			text = re.ReplaceAllString(text, "")
		}
		if err := os.WriteFile(jsPath, []byte(text), 0644); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 200, "msg": "Success", "data": nil})
	}
}
