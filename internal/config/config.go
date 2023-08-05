package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	SecretKey    string
	Account      string
	AccessExpire int64 `json:",default=86000"`
}
