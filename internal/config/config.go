package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	SecretKey    string `json:",string"`
	AccessExpire int64  `json:",default=86000"`
}

var (
	Version   string
	BuildDate string
)
