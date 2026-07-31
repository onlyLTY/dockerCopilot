package icons

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repository, err := normalizeRepository(r.URL.Query().Get("imageName"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		result, err := withIconConfigLock(func() (bool, error) {
			icons, err := readIcons(iconConfigPath())
			if err != nil {
				return false, err
			}
			removed := false
			files := make([]string, 0)
			for _, key := range matchingKeys(icons, repository) {
				files = append(files, filepath.Base(icons[key]))
				delete(icons, key)
				removed = true
			}
			if !removed {
				return false, nil
			}
			if err := writeIcons(iconConfigPath(), icons); err != nil {
				return false, err
			}
			for _, file := range files {
				if !iconFileReferenced(icons, file) {
					if err := os.Remove(filepath.Join(iconDirectory(), file)); err != nil && !os.IsNotExist(err) {
						return false, fmt.Errorf("删除图标文件失败: %v", err)
					}
				}
			}
			return true, nil
		})
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		_ = result
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"code": 200, "msg": "Success", "data": nil})
	}
}
