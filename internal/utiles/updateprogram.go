package utiles

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const maxUpdateDownload = 256 << 20

var releaseVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func UpdateProgram(ctx *svc.ServiceContext) error {
	githubProxy := strings.TrimRight(os.Getenv("githubProxy"), "/")
	prefix := ""
	if githubProxy != "" {
		prefix = githubProxy + "/"
	}
	versionURL := prefix + "https://raw.githubusercontent.com/onlyLTY/dockerCopilot/UGREEN/version"
	releaseBaseURL := prefix + "https://github.com/onlyLTY/dockerCopilot/releases/download"
	client := &http.Client{Timeout: 30 * time.Second}
	versionResp, err := client.Get(versionURL)
	if err != nil {
		return fmt.Errorf("获取版本信息失败: %w", err)
	}
	defer versionResp.Body.Close()
	if versionResp.StatusCode != http.StatusOK {
		return fmt.Errorf("版本服务返回状态 %s", versionResp.Status)
	}
	versionData, err := io.ReadAll(io.LimitReader(versionResp.Body, 1024))
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(versionData))
	if !releaseVersionPattern.MatchString(version) {
		return fmt.Errorf("版本号格式不合法")
	}

	downloadURL := fmt.Sprintf("%s/%s/dockerCopilot-%s.tar.gz", releaseBaseURL, version, runtime.GOARCH)
	tmp, err := os.CreateTemp("", "dockercopilot-update-*.tar.gz")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := downloadFile(client, downloadURL, tmp); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	extractDir, err := os.MkdirTemp("", "dockercopilot-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)
	if err := decompressTarGz(tmpPath, extractDir); err != nil {
		return err
	}
	binary, err := findUpdateBinary(extractDir)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return fmt.Errorf("更新程序为空")
	}
	return os.WriteFile("dockerCopilot-new", content, 0755)
}

func downloadFile(client *http.Client, url string, out *os.File) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载服务返回状态 %s", resp.Status)
	}
	if resp.ContentLength > maxUpdateDownload {
		return fmt.Errorf("更新包超过大小限制")
	}
	_, err = io.Copy(out, io.LimitReader(resp.Body, maxUpdateDownload+1))
	if err != nil {
		return err
	}
	info, err := out.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxUpdateDownload {
		return fmt.Errorf("更新包超过大小限制")
	}
	return nil
}

func decompressTarGz(gzFilePath, dest string) error {
	file, err := os.Open(gzFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Name == "" || filepath.IsAbs(header.Name) || strings.Contains(header.Name, ".."+string(filepath.Separator)) || strings.Contains(header.Name, "../") || strings.Contains(header.Name, `..\`) {
			return fmt.Errorf("归档条目路径不合法")
		}
		clean := filepath.Clean(header.Name)
		target := filepath.Join(dest, clean)
		if err := ensureUpdatePath(dest, target); err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0750); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(outFile, io.LimitReader(tarReader, maxUpdateDownload+1))
			closeErr := outFile.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("不支持的归档条目类型: %s", header.Name)
		}
	}
	return nil
}

func ensureUpdatePath(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("归档路径越界")
	}
	return nil
}

func findUpdateBinary(root string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "dockerCopilot-new" {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("更新包中缺少 dockerCopilot-new")
	}
	return found, nil
}
