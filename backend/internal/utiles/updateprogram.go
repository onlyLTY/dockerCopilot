package utiles

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

const maxUpdateDownload = 256 << 20

var releaseVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// 官方下载源 host 白名单（未走 proxy 时最终 URL 必须落在这些 host 上）。
var officialUpdateHosts = map[string]struct{}{
	"github.com":                {},
	"raw.githubusercontent.com": {},
}

func UpdateProgram(ctx *svc.ServiceContext) error {
	versionURL, err := OfficialVersionURL()
	if err != nil {
		return err
	}
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

	arch := runtime.GOARCH
	downloadURL, err := OfficialReleaseAssetURL(version, fmt.Sprintf("dockerCopilot-%s.tar.gz", arch))
	if err != nil {
		return err
	}
	checksumURL, err := OfficialReleaseAssetURL(version, fmt.Sprintf("dockerCopilot-%s.tar.gz.sha256", arch))
	if err != nil {
		return err
	}

	expectedSHA, err := fetchReleaseSHA256(client, checksumURL)
	if err != nil {
		return fmt.Errorf("获取更新包校验和失败: %w", err)
	}

	tmp, err := os.CreateTemp("", "dockercopilot-update-*.tar.gz")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := downloadFile(client, downloadURL, tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	actualSHA, err := fileSHA256(tmpPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actualSHA, expectedSHA) {
		return fmt.Errorf("更新包 SHA256 校验失败")
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

	// 若更新包内含前端资源（web 目录），先落到 ./web-new，
	// 由 start.sh 在下次重启时与二进制一起切换，保证前后端版本同步。
	// 旧版更新包（仅二进制）无 web 目录时跳过，保持向后兼容。
	if webSrc := findUpdateWebDir(extractDir); webSrc != "" {
		if err := os.RemoveAll("web-new"); err != nil {
			return err
		}
		if err := copyDir(webSrc, "web-new"); err != nil {
			return fmt.Errorf("复制前端资源失败: %w", err)
		}
	}

	// 二进制最后写：它是 start.sh 触发切换的信号。
	return os.WriteFile("dockerCopilot-new", content, 0755)
}

// OfficialVersionURL 返回（可选 githubProxy 前缀后的）官方 version 文件 URL。
func OfficialVersionURL() (string, error) {
	official := "https://raw.githubusercontent.com/onlyLTY/dockerCopilot/latest/version"
	return withGithubProxy(os.Getenv("githubProxy"), official)
}

// OfficialReleaseAssetURL 返回指定 release 资产的官方下载 URL（可经 githubProxy）。
func OfficialReleaseAssetURL(version, asset string) (string, error) {
	if !releaseVersionPattern.MatchString(version) {
		return "", fmt.Errorf("版本号格式不合法")
	}
	rawAsset := strings.TrimSpace(asset)
	// 拒绝路径分隔与 ..，避免依赖 filepath.Base 在各平台上的剥离行为。
	if rawAsset == "" || strings.Contains(rawAsset, "..") || strings.ContainsAny(rawAsset, `/\:`) {
		return "", fmt.Errorf("更新资产名不合法")
	}
	asset = filepath.Base(rawAsset)
	if asset == "" || asset != rawAsset {
		return "", fmt.Errorf("更新资产名不合法")
	}
	official := fmt.Sprintf("https://github.com/onlyLTY/dockerCopilot/releases/download/%s/%s", version, asset)
	return withGithubProxy(os.Getenv("githubProxy"), official)
}

// withGithubProxy 仅允许「代理前缀 + 官方 https URL」形式；官方 host 必须在白名单内。
// githubProxy 为空时直接返回官方 URL。代理本身不得改写官方 path/host 语义。
func withGithubProxy(githubProxy, officialURL string) (string, error) {
	if err := assertOfficialUpdateURL(officialURL); err != nil {
		return "", err
	}
	proxy := strings.TrimSpace(githubProxy)
	if proxy == "" {
		return officialURL, nil
	}
	proxy = strings.TrimRight(proxy, "/")
	parsed, err := url.Parse(proxy)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("githubProxy 不是合法 URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("githubProxy 仅支持 http/https")
	}
	// 禁止在 proxy 里夹带 userinfo，避免凭据进日志/环境
	if parsed.User != nil {
		return "", fmt.Errorf("githubProxy 不支持账号密码")
	}
	final := proxy + "/" + officialURL
	// 最终 URL 必须仍以官方 URL 为后缀，防止 proxy 值本身吞掉/改写目标
	if !strings.HasSuffix(final, "/"+officialURL) && !strings.HasSuffix(final, officialURL) {
		return "", fmt.Errorf("githubProxy 无法安全拼接到官方地址")
	}
	if _, err := url.ParseRequestURI(final); err != nil {
		return "", fmt.Errorf("更新地址不合法")
	}
	return final, nil
}

func assertOfficialUpdateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("官方更新地址解析失败")
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("官方更新地址必须使用 https")
	}
	host := strings.ToLower(u.Hostname())
	if _, ok := officialUpdateHosts[host]; !ok {
		return fmt.Errorf("官方更新地址 host 不在白名单")
	}
	path := u.EscapedPath()
	switch host {
	case "raw.githubusercontent.com":
		if path != "/onlyLTY/dockerCopilot/latest/version" {
			return fmt.Errorf("官方 version 路径不合法")
		}
	case "github.com":
		// /onlyLTY/dockerCopilot/releases/download/<ver>/<asset>
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) != 6 ||
			parts[0] != "onlyLTY" ||
			parts[1] != "dockerCopilot" ||
			parts[2] != "releases" ||
			parts[3] != "download" ||
			!releaseVersionPattern.MatchString(parts[4]) ||
			parts[5] == "" || strings.Contains(parts[5], "..") {
			return fmt.Errorf("官方 release 资产路径不合法")
		}
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("官方更新地址包含非法附加信息")
	}
	return nil
}

func fetchReleaseSHA256(client *http.Client, checksumURL string) (string, error) {
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("校验和文件返回状态 %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	sum, err := parseSHA256File(string(data))
	if err != nil {
		return "", err
	}
	return sum, nil
}

// parseSHA256File 接受 `sha256sum` 行（`<hex>  filename` / `<hex> *filename`）或纯 64 位 hex。
func parseSHA256File(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("校验和文件为空")
	}
	// 只取第一行有效内容
	line := content
	if i := strings.IndexAny(content, "\r\n"); i >= 0 {
		line = strings.TrimSpace(content[:i])
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", fmt.Errorf("校验和文件格式不合法")
	}
	sum := strings.ToLower(fields[0])
	if len(sum) != 64 {
		return "", fmt.Errorf("校验和长度不合法")
	}
	for _, c := range sum {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", fmt.Errorf("校验和不是合法十六进制")
		}
	}
	return sum, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(f, maxUpdateDownload+1)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// findUpdateWebDir 在解压目录中查找名为 web 的目录，返回其路径；未找到返回空串。
func findUpdateWebDir(root string) string {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "web" {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

// copyDir 递归复制目录，用于将更新包中的前端资源落到 web-new。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0750)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func downloadFile(client *http.Client, rawURL string, out *os.File) error {
	resp, err := client.Get(rawURL)
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
