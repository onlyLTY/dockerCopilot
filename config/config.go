package config

import "github.com/zeromicro/go-zero/core/logx"

type Config struct {
	Log  logx.LogConf
	Host string `json:",default=127.0.0.1"`
	Port int    `json:",default=8060"`
}
