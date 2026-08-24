package icons

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/imageref"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	maxImageFileSize     int64 = 2 << 20
	maxUploadRequestSize int64 = maxImageFileSize + (1 << 20)
	maxImageNameLength         = 255
)

var imageLogosMu sync.Mutex

var allowedImageTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequestSize)
		// #nosec G120 -- MaxBytesReader enforces a strict limit on the complete request body.
		err := r.ParseMultipartForm(1 << 20)
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				writeUploadError(w, http.StatusRequestEntityTooLarge, "upload exceeds 2MB limit")
				return
			}
			writeUploadError(w, http.StatusBadRequest, "failed to parse form")
			return
		}
		defer r.MultipartForm.RemoveAll()

		// 2. 获取文件和 Key
		file, handler, err := r.FormFile("file")
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, "failed to get file")
			return
		}
		defer file.Close()
		if handler.Size > maxImageFileSize {
			writeUploadError(w, http.StatusRequestEntityTooLarge, "upload exceeds 2MB limit")
			return
		}

		imageNameKey := r.FormValue("imageName")
		if err := validateImageName(imageNameKey); err != nil {
			writeUploadError(w, http.StatusBadRequest, err.Error())
			return
		}

		// 3. 确保目录存在 (防御性编程)
		dataPath := imageUploadDir
		if err := os.MkdirAll(dataPath, 0o700); err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to prepare upload dir")
			return
		}
		dataRoot, err := os.OpenRoot(dataPath)
		if err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to open upload dir")
			return
		}
		defer dataRoot.Close()

		// 4. 确定文件名
		filename, err := generateStoredFilename(file, handler)
		if err != nil {
			writeUploadError(w, http.StatusBadRequest, err.Error())
			return
		}

		dst, err := dataRoot.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to create file on server")
			return
		}
		copySucceeded := false
		defer func() {
			_ = dst.Close()
			if !copySucceeded {
				_ = dataRoot.Remove(filename)
			}
		}()

		written, err := io.Copy(dst, io.LimitReader(file, maxImageFileSize+1))
		if err != nil || written > maxImageFileSize {
			if written > maxImageFileSize {
				writeUploadError(w, http.StatusRequestEntityTooLarge, "upload exceeds 2MB limit")
				return
			}
			writeUploadError(w, http.StatusInternalServerError, "failed to copy file content")
			return
		}
		if err := dst.Sync(); err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to persist file content")
			return
		}
		if err := dst.Close(); err != nil {
			writeUploadError(w, http.StatusInternalServerError, "failed to close file")
			return
		}
		copySucceeded = true

		// 5. 更新 JSON 映射，并清理被替换的旧文件。
		oldFilename, err := updateImageLogoMapping(imageNameKey, filename)
		if err != nil {
			_ = dataRoot.Remove(filename)
			writeUploadError(w, http.StatusInternalServerError, "failed to update config")
			return
		}
		if oldFilename != "" && oldFilename != filename {
			_ = dataRoot.Remove(oldFilename)
		}

		httpx.OkJsonCtx(r.Context(), w, types.Resp{
			Code: 200,
			Msg:  "Success",
			Data: filename,
		})
	}
}

func validateImageName(imageName string) error {
	imageName = strings.TrimSpace(imageName)
	if imageName == "" || len(imageName) > maxImageNameLength {
		return fmt.Errorf("imageName is required")
	}
	if _, err := imageref.RepositoryKey(imageName); err != nil {
		return fmt.Errorf("invalid imageName")
	}
	return nil
}

func generateStoredFilename(file multipart.File, handler *multipart.FileHeader) (string, error) {
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to inspect upload")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset upload stream")
	}

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	expectedType, ok := allowedImageTypes[ext]
	if !ok {
		return "", fmt.Errorf("only png, jpg, jpeg, webp and gif files are allowed")
	}

	detectedType := http.DetectContentType(header[:n])
	if detectedType != expectedType {
		return "", fmt.Errorf("uploaded file content does not match its extension")
	}

	return uuid.NewString() + ext, nil
}

func writeUploadError(w http.ResponseWriter, statusCode int, msg string) {
	httpx.WriteJson(w, statusCode, types.Resp{
		Code: statusCode,
		Msg:  msg,
		Data: map[string]interface{}{},
	})
}
