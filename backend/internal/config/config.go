package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Auth struct { // JWT 认证需要的密钥和过期时间配置
		AccessSecret string
		AccessExpire int64
	}
	CorsOrigins []string
	Compose     ComposeConfig
}

type ComposeConfig struct {
	ScanPaths         []string
	BackupDir         string
	PathMappings      []ComposePathMapping
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

// Version / BuildDate 默认值用于本地或未通过 -ldflags 注入的构建，
// 避免前端一直显示“读取中”。正式构建时会被 -X 覆盖为真实版本。
var (
	Version   = "dev"
	BuildDate = "unknown"
)
