package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	// DataDir 应用数据根目录（设置/备份/图标/任务进度）。
	// 可省略；优先级：环境变量 DATA_DIR > 本字段 > 默认 /data。
	// 相对路径相对进程工作目录解析。容器内通常不写，挂载 /data 即可。
	//lint:ignore SA5008 go-zero uses optional as a configuration tag.
	DataDir string   `json:",optional"`
	Auth    struct { // JWT 认证需要的密钥和过期时间配置
		AccessSecret string
		AccessExpire int64
	}
	CorsOrigins []string
	Compose     ComposeConfig
	// PullTimeoutSec 拉取镜像超时（秒），默认 300（5 分钟）。
	// 适用于单容器更新、恢复和 Compose 部署中的镜像拉取；零或负值使用默认值。
	//lint:ignore SA5008 go-zero uses optional as a configuration tag.
	PullTimeoutSec int64 `json:",optional"`
}

// MinAccessSecretLen secretKey / AccessSecret 最短长度（与 go-zero JWT 下限及文案「8 位以上」一致）。
const MinAccessSecretLen = 8

// DefaultPullTimeoutSec 单容器更新/恢复拉取镜像的默认超时（秒）。
const DefaultPullTimeoutSec int64 = 300

// ValidateAccessSecret 启动时校验登录/JWT 密钥：至少 8 位，且不能为纯数字。
// 与历史提示「非纯数字且大于八位」对齐；「8 位以上」按常见语义含 8 位（len >= 8）。
func ValidateAccessSecret(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("secretKey 不能为空，要求至少 %d 位且不能为纯数字", MinAccessSecretLen)
	}
	if len(secret) < MinAccessSecretLen {
		return fmt.Errorf("secretKey 长度不足：至少 %d 位，当前 %d 位", MinAccessSecretLen, len(secret))
	}
	if isAllDigits(secret) {
		return fmt.Errorf("secretKey 不能为纯数字，请混用字母或其它字符")
	}
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

type ComposeConfig struct {
	ScanPaths []string
	// BackupDir 可省略：空时使用 {DataDir}/backups/compose-projects（或 BACKUP_DIR/compose-projects）。
	// 仅手动备份 Compose 项目文件；容器备份页走 {DataDir}/backups。
	//lint:ignore SA5008 go-zero uses optional as a configuration tag.
	BackupDir string `json:",optional"`
	//lint:ignore SA5008 go-zero uses optional as a configuration tag.
	PathMappings      []ComposePathMapping `json:",optional"`
	MaxDepth          int
	MaxFileSize       int64
	MaxFiles          int
	AllowHighRisk     bool
	CommandTimeoutSec int64
}

type ComposePathMapping struct {
	HostPath      string
	ContainerPath string
}

// Version / BuildDate 默认值用于本地或未通过 -ldflags 注入的构建。
// 正式构建时会被 -X 覆盖；本地启动若仍是 dev，会尝试读取仓库根目录的 version 文件。
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// ResolveVersion 在启动时解析版本号：
// 1. 已由 -ldflags 注入（非 dev）→ 保持不变
// 2. 否则读 version 文件（可用 VERSION_FILE 指定路径）
// 3. 文件不存在或为空 → 保留 dev
func ResolveVersion() {
	if strings.TrimSpace(Version) != "" && Version != "dev" {
		return
	}
	path := strings.TrimSpace(os.Getenv("VERSION_FILE"))
	candidates := make([]string, 0, 6)
	if path != "" {
		candidates = append(candidates, path)
	}
	candidates = append(candidates, "version", filepath.Join("..", "version"))
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "version"),
			filepath.Join(dir, "..", "version"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "version"),
			filepath.Join(wd, "..", "version"),
		)
	}

	seen := map[string]struct{}{}
	for _, p := range candidates {
		p = filepath.Clean(p)
		if p == "" || p == "." {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(raw))
		// 只取第一行，避免 BOM/换行干扰
		if i := strings.IndexAny(v, "\r\n"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		if v == "" {
			continue
		}
		Version = v
		if BuildDate == "" || BuildDate == "unknown" {
			if info, err := os.Stat(p); err == nil {
				BuildDate = info.ModTime().Local().Format(time.RFC1123)
			}
		}
		return
	}
}
