package icons

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// UploadInput multipart 解析后的 DTO。
type UploadInput struct {
	ImageName    string
	OriginalName string
	File         io.ReadSeekCloser
}

// UploadError 带 HTTP 状态的业务错误（图标上传历史契约：status == body.code）。
type UploadError struct {
	Status int
	Msg    string
}

func (e *UploadError) Error() string { return e.Msg }

type UploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadLogic {
	return &UploadLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UploadLogic) Upload(in *UploadInput) (*types.Resp, error) {
	if in == nil || in.File == nil {
		return nil, &UploadError{Status: http.StatusBadRequest, Msg: "failed to get file"}
	}
	repository, err := NormalizeRepository(in.ImageName)
	if err != nil {
		return nil, &UploadError{Status: http.StatusBadRequest, Msg: err.Error()}
	}
	filename, err := generateStoredFilename(in.File, in.OriginalName)
	if err != nil {
		return nil, &UploadError{Status: http.StatusBadRequest, Msg: err.Error()}
	}
	if err := os.MkdirAll(iconDirectory(), 0755); err != nil {
		return nil, &UploadError{Status: http.StatusInternalServerError, Msg: "failed to prepare upload dir"}
	}

	dstPath := filepath.Join(iconDirectory(), filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, &UploadError{Status: http.StatusInternalServerError, Msg: "failed to create file on server"}
	}
	if _, err := io.Copy(dst, in.File); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return nil, &UploadError{Status: http.StatusInternalServerError, Msg: "failed to copy file content"}
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return nil, &UploadError{Status: http.StatusInternalServerError, Msg: "failed to close uploaded file"}
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
		return nil, &UploadError{Status: http.StatusInternalServerError, Msg: "failed to update config"}
	}
	return &types.Resp{Code: 200, Msg: "Success", Data: result}, nil
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
