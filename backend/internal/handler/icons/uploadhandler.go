package icons

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to parse form: %v", err))
			return
		}
		file, handler, err := r.FormFile("file")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to get file: %v", err))
			return
		}
		defer file.Close()

		repository, err := normalizeRepository(r.FormValue("imageName"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		filename, err := iconFilename(repository, handler.Filename)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := os.MkdirAll(iconDirectory(), 0755); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		dstPath := filepath.Join(iconDirectory(), filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to create file on server: %v", err))
			return
		}
		if _, err := io.Copy(dst, file); err != nil {
			dst.Close()
			os.Remove(dstPath)
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to copy file content: %v", err))
			return
		}
		if err := dst.Close(); err != nil {
			os.Remove(dstPath)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		result, err := withIconConfigLock(func() (string, error) {
			icons, err := readIcons(iconConfigPath())
			if err != nil {
				return "", err
			}
			oldFiles := make([]string, 0)
			for _, key := range matchingKeys(icons, repository) {
				if old := filepath.Base(icons[key]); old != filename {
					oldFiles = append(oldFiles, old)
				}
				delete(icons, key)
			}
			icons[repository] = imageURL(filename)
			if err := writeIcons(iconConfigPath(), icons); err != nil {
				return "", err
			}
			for _, old := range oldFiles {
				if !iconFileReferenced(icons, old) {
					_ = os.Remove(filepath.Join(iconDirectory(), old))
				}
			}
			return filename, nil
		})
		if err != nil {
			os.Remove(dstPath)
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to update config: %v", err))
			return
		}

		httpx.OkJsonCtx(r.Context(), w, types.Resp{Code: 200, Msg: "Success", Data: result})
	}
}

func iconFileReferenced(icons map[string]string, filename string) bool {
	for _, value := range icons {
		if filepath.Base(value) == filename {
			return true
		}
	}
	return false
}
