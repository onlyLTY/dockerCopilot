package icons

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			writeUploadError(w, http.StatusBadRequest, "failed to parse form")
			return
		}
		file, handler, err := r.FormFile("file")
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, "failed to get file")
			return
		}
		defer file.Close()

		repository, err := normalizeRepository(r.FormValue("imageName"))
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, err.Error())
			return
		}
		filename, err := generateStoredFilename(file, handler.Filename)
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := os.MkdirAll(iconDirectory(), 0755); err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to prepare upload dir")
			return
		}

		dstPath := filepath.Join(iconDirectory(), filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to create file on server")
			return
		}
		if _, err := io.Copy(dst, file); err != nil {
			_ = dst.Close()
			_ = os.Remove(dstPath)
			writeUploadError(w, http.StatusInternalServerError, "failed to copy file content")
			return
		}
		if err := dst.Close(); err != nil {
			_ = os.Remove(dstPath)
			writeUploadError(w, http.StatusInternalServerError, "failed to close uploaded file")
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
			_ = os.Remove(dstPath)
			writeUploadError(w, http.StatusInternalServerError, "failed to update config")
			return
		}

		httpx.OkJsonCtx(r.Context(), w, types.Resp{Code: 200, Msg: "Success", Data: result})
	}
}

func generateStoredFilename(file io.ReadSeeker, original string) (string, error) {
	ext, err := iconExtension(original)
	if err != nil {
		return "", err
	}

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to inspect upload")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset upload stream")
	}

	if ext == ".svg" {
		if !strings.Contains(strings.ToLower(string(header[:n])), "<svg") {
			return "", fmt.Errorf("uploaded file content does not match its extension")
		}
	} else {
		expected := map[string]string{
			".png":  "image/png",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".webp": "image/webp",
			".gif":  "image/gif",
		}[ext]
		if detected := http.DetectContentType(header[:n]); detected != expected {
			return "", fmt.Errorf("uploaded file content does not match its extension")
		}
	}

	return uuid.NewString() + ext, nil
}

func writeUploadError(w http.ResponseWriter, statusCode int, msg string) {
	httpx.WriteJson(w, statusCode, types.Resp{Code: statusCode, Msg: msg, Data: map[string]interface{}{}})
}

func iconFileReferenced(icons map[string]string, filename string) bool {
	for _, value := range icons {
		if filepath.Base(value) == filename {
			return true
		}
	}
	return false
}
