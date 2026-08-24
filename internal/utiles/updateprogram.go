package utiles

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	maxUpdateArchiveSize   int64 = 256 << 20
	maxUpdateExtractedSize int64 = 512 << 20
	maxChecksumFileSize    int64 = 4 << 10
)

var ErrAlreadyLatest = errors.New("当前已是最新版本")

func UpdateProgram(ctx context.Context) error {
	version, err := GetRemoteVersion(ctx)
	if err != nil {
		return fmt.Errorf("获取最新版本失败: %w", err)
	}
	if version == currentVersion() {
		return ErrAlreadyLatest
	}

	releaseBaseURL, err := githubURL("https://github.com/onlyLTY/dockerCopilot/releases/download")
	if err != nil {
		return err
	}
	archiveName := fmt.Sprintf("dockerCopilot-%s.tar.gz", runtime.GOARCH)
	downloadURL := fmt.Sprintf("%s/%s/%s", strings.TrimRight(releaseBaseURL, "/"), url.PathEscape(version), archiveName)
	checksumURL := downloadURL + ".sha256"

	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("定位当前程序失败: %w", err)
	}
	appDir := filepath.Dir(executablePath)
	stagingDir, err := os.MkdirTemp(appDir, ".dockercopilot-update-*")
	if err != nil {
		return fmt.Errorf("创建更新暂存目录失败: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	httpClient := newHTTPSClient(2 * time.Minute)
	archivePath := filepath.Join(stagingDir, archiveName)
	checksumPath := archivePath + ".sha256"
	if err := downloadFile(ctx, httpClient, downloadURL, archivePath, maxUpdateArchiveSize); err != nil {
		return fmt.Errorf("下载更新包失败: %w", err)
	}
	if err := downloadFile(ctx, httpClient, checksumURL, checksumPath, maxChecksumFileSize); err != nil {
		return fmt.Errorf("下载更新校验文件失败: %w", err)
	}
	if err := verifySHA256(archivePath, checksumPath); err != nil {
		return fmt.Errorf("更新包校验失败: %w", err)
	}

	extractDir := filepath.Join(stagingDir, "extract")
	if err := os.Mkdir(extractDir, 0o700); err != nil {
		return fmt.Errorf("创建解压目录失败: %w", err)
	}
	if err := decompressTarGz(archivePath, extractDir); err != nil {
		return fmt.Errorf("解压更新包失败: %w", err)
	}

	newBinary := filepath.Join(extractDir, "dockerCopilot-new")
	info, err := os.Lstat(newBinary)
	if err != nil {
		return fmt.Errorf("更新包缺少 dockerCopilot-new: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return errors.New("更新程序不是有效的普通文件")
	}
	if err := validateLinuxBinary(newBinary); err != nil {
		return err
	}
	// #nosec G302 -- the staged application must be executable; 0700 restricts it to the service user.
	if err := os.Chmod(newBinary, 0o700); err != nil {
		return fmt.Errorf("设置更新程序权限失败: %w", err)
	}

	pendingPath := filepath.Join(appDir, "dockerCopilot-new")
	if err := os.Remove(pendingPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理旧更新文件失败: %w", err)
	}
	if err := os.Rename(newBinary, pendingPath); err != nil {
		return fmt.Errorf("安装更新文件失败: %w", err)
	}
	return nil
}

func downloadFile(ctx context.Context, client *http.Client, rawURL, dest string, maxBytes int64) (retErr error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return errors.New("仅允许有效的 HTTPS 下载地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回 %s", resp.Status)
	}
	if resp.ContentLength > maxBytes {
		return errors.New("响应大小超过限制")
	}

	destination, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	destinationRoot, err := os.OpenRoot(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer destinationRoot.Close()
	destinationName := filepath.Base(destination)
	out, err := destinationRoot.OpenFile(destinationName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); retErr == nil && closeErr != nil {
			retErr = closeErr
		}
		if retErr != nil {
			_ = destinationRoot.Remove(destinationName)
		}
	}()

	written, err := io.Copy(out, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		return errors.New("响应大小超过限制")
	}
	return nil
}

func verifySHA256(archivePath, checksumPath string) error {
	checksum, checksumRoot, err := openFileWithinParent(checksumPath)
	if err != nil {
		return err
	}
	defer checksumRoot.Close()
	defer checksum.Close()
	checksumData, err := io.ReadAll(io.LimitReader(checksum, maxChecksumFileSize+1))
	if err != nil {
		return err
	}
	if int64(len(checksumData)) > maxChecksumFileSize {
		return errors.New("SHA-256 文件超过大小限制")
	}
	fields := strings.Fields(string(checksumData))
	if len(fields) == 0 || len(fields[0]) != sha256.Size*2 {
		return errors.New("SHA-256 文件格式错误")
	}
	expected, err := hex.DecodeString(fields[0])
	if err != nil {
		return errors.New("SHA-256 文件格式错误")
	}

	archive, archiveRoot, err := openFileWithinParent(archivePath)
	if err != nil {
		return err
	}
	defer archiveRoot.Close()
	defer archive.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return err
	}
	if !equalBytes(hash.Sum(nil), expected) {
		return errors.New("SHA-256 不匹配")
	}
	return nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for index := range a {
		result |= a[index] ^ b[index]
	}
	return result == 0
}

func decompressTarGz(gzFilePath, dest string) error {
	file, archiveRoot, err := openFileWithinParent(gzFilePath)
	if err != nil {
		return err
	}
	defer archiveRoot.Close()
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	destinationRoot, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer destinationRoot.Close()
	tarReader := tar.NewReader(gzr)
	var extracted int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Size < 0 || header.Size > maxUpdateExtractedSize-extracted {
			return errors.New("解压内容超过大小限制")
		}
		extracted += header.Size

		cleanName := filepath.Clean(filepath.FromSlash(header.Name))
		if cleanName == "." || filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("压缩包包含非法路径 %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := destinationRoot.MkdirAll(cleanName, 0o700); err != nil {
				return err
			}
		case tar.TypeReg, byte(0):
			if err := destinationRoot.MkdirAll(filepath.Dir(cleanName), 0o700); err != nil {
				return err
			}
			outFile, err := destinationRoot.OpenFile(cleanName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(outFile, tarReader, header.Size)
			closeErr := outFile.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("压缩包包含不支持的文件类型 %d", header.Typeflag)
		}
	}
}

func openFileWithinParent(path string) (*os.File, *os.Root, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(absolutePath))
	if err != nil {
		return nil, nil, err
	}
	file, err := root.Open(filepath.Base(absolutePath))
	if err != nil {
		_ = root.Close()
		return nil, nil, err
	}
	return file, root, nil
}

func validateLinuxBinary(path string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	binary, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("更新程序不是有效的 ELF 文件: %w", err)
	}
	defer binary.Close()
	expectedMachine := map[string]elf.Machine{
		"amd64": elf.EM_X86_64,
		"arm64": elf.EM_AARCH64,
	}[runtime.GOARCH]
	if expectedMachine == elf.EM_NONE || binary.Machine != expectedMachine {
		return fmt.Errorf("更新程序架构不匹配: %s", binary.Machine)
	}
	return nil
}
