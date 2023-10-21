package svc

import (
	"github.com/flosch/pongo2"
	"github.com/google/uuid"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/config"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/middleware"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/module"
	"github.com/zeromicro/go-zero/rest"
)

type ServiceContext struct {
	Config                     config.Config
	CookieCheckMiddleware      rest.Middleware
	Jwtuuid                    string
	BearerTokenCheckMiddleware rest.Middleware
	JwtSecret                  string
	Template                   *pongo2.TemplateSet
	PortainerJwt               string
	HubImageInfo               *module.ImageUpdateData
	IndexCheckMiddleware       rest.Middleware
	ProgressStore              ProgressStoreType
}

type TaskProgress struct {
	Percentage int
	Message    string
	IsDone     bool
}

type ProgressStoreType map[string]TaskProgress

func NewServiceContext(c config.Config, loaders *loader.Loader) *ServiceContext {
	uuidtmp := uuid.New().String()
	jwtSecret := c.SecretKey
	return &ServiceContext{
		Config:                     c,
		CookieCheckMiddleware:      middleware.NewCookieCheckMiddleware(uuidtmp).Handle,
		Jwtuuid:                    uuidtmp,
		BearerTokenCheckMiddleware: middleware.NewBearerTokenCheckMiddleware(jwtSecret).Handle,
		JwtSecret:                  jwtSecret,
		Template:                   pongo2.NewSet("", loaders),
		HubImageInfo:               module.NewImageCheck(),
		IndexCheckMiddleware:       middleware.NewIndexCheckMiddleware(uuidtmp).Handle,
		ProgressStore:              make(ProgressStoreType),
	}
}
