package svc

import "github.com/onlyLTY/oneKeyUpdate/v2/config"

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
