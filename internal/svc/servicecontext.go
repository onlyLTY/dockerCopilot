package svc

import (
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"sync"
	"time"
)

type ServiceContext struct {
	Config                     config.Config
	CookieCheckMiddleware      rest.Middleware
	Jwtuuid                    string
	BearerTokenCheckMiddleware rest.Middleware
	JwtSecret                  string
	PortainerJwt               string
	HubImageInfo               *module.ImageUpdateData
	IndexCheckMiddleware       rest.Middleware
	ProgressStore              ProgressStoreType
	DockerClient               *client.Client
	mu                         sync.RWMutex
	activeContainerUpdates     map[string]struct{}
	activeRestore              bool
}

const (
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)

type TaskProgress struct {
	TaskID     string
	Percentage int
	Message    string
	Name       string
	DetailMsg  string
	IsDone     bool
	Status     string
	UpdatedAt  time.Time
}

type ProgressStoreType map[string]TaskProgress

func NewServiceContext(c config.Config) *ServiceContext {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logx.Errorf("Unable to create docker client: %s", err)
	}
	return &ServiceContext{
		Config:                 c,
		HubImageInfo:           module.NewImageCheck(),
		ProgressStore:          make(ProgressStoreType),
		DockerClient:           cli,
		activeContainerUpdates: make(map[string]struct{}),
	}
}

func (ctx *ServiceContext) UpdateProgress(taskID string, progress TaskProgress) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if progress.Status == "" {
		if progress.IsDone {
			progress.Status = TaskStatusCompleted
		} else {
			progress.Status = TaskStatusRunning
		}
	}
	progress.UpdatedAt = time.Now()
	ctx.ProgressStore[taskID] = progress
	if len(ctx.ProgressStore) > 1000 {
		ctx.cleanupProgressLocked(time.Hour)
	}
}

func (ctx *ServiceContext) GetProgress(taskID string) (TaskProgress, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	progress, ok := ctx.ProgressStore[taskID]
	return progress, ok
}

func (ctx *ServiceContext) CleanupProgress(retention time.Duration) int {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.cleanupProgressLocked(retention)
}

func (ctx *ServiceContext) cleanupProgressLocked(retention time.Duration) int {
	cutoff := time.Now().Add(-retention)
	removed := 0
	for taskID, progress := range ctx.ProgressStore {
		if progress.IsDone && !progress.UpdatedAt.IsZero() && progress.UpdatedAt.Before(cutoff) {
			delete(ctx.ProgressStore, taskID)
			removed++
		}
	}
	return removed
}

func (ctx *ServiceContext) BeginContainerUpdate(containerID string) bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.activeContainerUpdates == nil {
		ctx.activeContainerUpdates = make(map[string]struct{})
	}
	if _, exists := ctx.activeContainerUpdates[containerID]; exists {
		return false
	}
	ctx.activeContainerUpdates[containerID] = struct{}{}
	return true
}

func (ctx *ServiceContext) EndContainerUpdate(containerID string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	delete(ctx.activeContainerUpdates, containerID)
}

func (ctx *ServiceContext) BeginRestore() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.activeRestore {
		return false
	}
	ctx.activeRestore = true
	return true
}

func (ctx *ServiceContext) EndRestore() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.activeRestore = false
}
