package svc

import (
	"github.com/flosch/pongo2"
	"github.com/google/uuid"
	loader "github.com/nathan-osman/pongo2-embed-loader"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/config"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/middleware"
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/module"
	"github.com/zeromicro/go-zero/rest"
	"sync"
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
	mu                         sync.Mutex
}

type TaskProgress struct {
	TaskID     string
	Percentage int
	Message    string
	Name       string
	DetailMsg  string
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

func (ctx *ServiceContext) UpdateProgress(taskID string, progress TaskProgress) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.ProgressStore[taskID] = progress
}

func (ctx *ServiceContext) GetProgress(taskID string) (TaskProgress, bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	progress, ok := ctx.ProgressStore[taskID]
	return progress, ok
}
