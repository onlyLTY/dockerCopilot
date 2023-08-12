package svc

import (
	"github.com/flosch/pongo2"
	"github.com/google/uuid"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/config"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/middleware"
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/module"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config                config.Config
	CookieCheckMiddleware rest.Middleware
	Template              *pongo2.TemplateSet
	PortainerJwt          string
	HubImageInfo          *module.ImageUpdateData
	IndexCheckMiddleware  rest.Middleware
	Jwtuuid               string
}

func NewServiceContext(c config.Config, loaders *loader.Loader) *ServiceContext {
	uuidtmp := uuid.New().String()
	return &ServiceContext{
		Config:                c,
		CookieCheckMiddleware: middleware.NewCookieCheckMiddleware(uuidtmp).Handle,
		Template:              pongo2.NewSet("", loaders),
		HubImageInfo:          module.NewImageCheck(),
		Jwtuuid:               uuidtmp,
		IndexCheckMiddleware:  middleware.NewIndexCheckMiddleware(uuidtmp).Handle,
	}
}
