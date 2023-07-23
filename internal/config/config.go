package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Secret_key string
	Account    string
}
