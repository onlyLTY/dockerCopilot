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

var (
	Version   string
	BuildDate string
)
