package svc

import (
	"github.com/flosch/pongo2"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/config"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/middleware"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config                config.Config
	CookieCheckMiddleware rest.Middleware
	Template              *pongo2.TemplateSet
	PortainerJwt          string
}

func NewServiceContext(c config.Config, loaders *loader.Loader) *ServiceContext {
	return &ServiceContext{
		Config:                c,
		CookieCheckMiddleware: middleware.NewCookieCheckMiddleware().Handle,
		Template:              pongo2.NewSet("", loaders),
	}
}
