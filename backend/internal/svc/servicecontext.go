package svc

import (
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/robfig/cron/v3"
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
	mu                         sync.Mutex
	ComposeMu                  sync.Mutex
	ComposeTokens              map[string]ComposeToken

	// 定时任务：更新检查与自动备份各持有一个 cronTask，job 由 main 注入（避免 svc 反向依赖 utiles）
	updateTask cronTask
	backupTask cronTask
	cronMu     sync.Mutex
}

// cronTask 封装单个可动态重新调度的定时任务。
type cronTask struct {
	cron  *cron.Cron
	jobID cron.EntryID
	job   func()
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

type ComposeToken struct {
	ProjectID string
	Filename  string
	Version   string
	ExpiresAt time.Time
}

func NewServiceContext(c config.Config) *ServiceContext {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logx.Errorf("Unable to create docker client: %s", err)
	}
	return &ServiceContext{
		Config:        c,
		HubImageInfo:  module.NewImageCheck(),
		ProgressStore: make(ProgressStoreType),
		ComposeTokens: make(map[string]ComposeToken),
		DockerClient:  cli,
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

// StartUpdateCron 启动更新检查定时任务；job 为要执行的检查逻辑（由 main 注入）。
// spec 为 5 段 cron 表达式（分 时 日 月 周）。
func (ctx *ServiceContext) StartUpdateCron(spec string, job func()) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.startLocked(&ctx.updateTask, spec, job)
}

// RescheduleUpdateCron 按新的 cron 表达式重新调度更新检查任务。
func (ctx *ServiceContext) RescheduleUpdateCron(spec string) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.rescheduleLocked(&ctx.updateTask, spec)
}

// StartBackupCron 启动自动备份定时任务；job 为要执行的备份逻辑（由 main 注入）。
func (ctx *ServiceContext) StartBackupCron(spec string, job func()) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.startLocked(&ctx.backupTask, spec, job)
}

// RescheduleBackupCron 按新的 cron 表达式重新调度自动备份任务。
func (ctx *ServiceContext) RescheduleBackupCron(spec string) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.rescheduleLocked(&ctx.backupTask, spec)
}

// startLocked 首次初始化调度器并注入 job，然后按 spec 调度。需持有 cronMu。
func (ctx *ServiceContext) startLocked(t *cronTask, spec string, job func()) error {
	t.job = job
	if t.cron == nil {
		t.cron = cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
		)))
		t.cron.Start()
	}
	return ctx.scheduleLocked(t, spec)
}

// rescheduleLocked 在已初始化的前提下按新 spec 重新调度。需持有 cronMu。
func (ctx *ServiceContext) rescheduleLocked(t *cronTask, spec string) error {
	if t.cron == nil || t.job == nil {
		return nil
	}
	return ctx.scheduleLocked(t, spec)
}

// scheduleLocked 移除旧任务并按 spec 添加新任务；spec 为空表示关闭（仅移除）。需持有 cronMu。
func (ctx *ServiceContext) scheduleLocked(t *cronTask, spec string) error {
	if t.jobID != 0 {
		t.cron.Remove(t.jobID)
		t.jobID = 0
	}
	if spec == "" {
		return nil
	}
	id, err := t.cron.AddFunc(spec, t.job)
	if err != nil {
		return err
	}
	t.jobID = id
	return nil
}
