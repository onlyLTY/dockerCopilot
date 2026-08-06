package svc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/errorx"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config               config.Config
	HubImageInfo         *module.ImageUpdateData
	ProgressStore        ProgressStoreType
	progressPath         string
	progressPersistTimer *time.Timer
	DockerClient         *client.Client
	mu                   sync.Mutex
	ComposeMu            sync.Mutex
	ComposeTokens        map[string]ComposeToken
	// composeOps 按项目 ID 互斥：同一项目同时只允许一个 deploy/cleanup 进行中。
	composeOpsMu       sync.Mutex
	composeOps         map[string]string
	updateMu           sync.Mutex
	updatingContainers map[string]string

	// 定时任务：更新检查与自动备份各持有一个 cronTask，job 由 main 注入（避免 svc 反向依赖 utiles）
	updateTask cronTask
	backupTask cronTask
	cronMu     sync.Mutex
}

// RequireDocker 在调用 Engine API 前检查客户端；不可用时返回 errorx.ErrDockerUnavailable（业务码 503）。
func (ctx *ServiceContext) RequireDocker() error {
	if ctx == nil || ctx.DockerClient == nil {
		return errorx.ErrDockerUnavailable
	}
	return nil
}

// HasDocker 是否已持有可用的 Docker 客户端（不代表此刻一定能 Ping 通）。
func (ctx *ServiceContext) HasDocker() bool {
	return ctx != nil && ctx.DockerClient != nil
}

const (
	// progressPersistDelay 进行中任务合并写盘间隔，降低高频进度更新的磁盘压力。
	progressPersistDelay = 300 * time.Millisecond
	// maxDoneProgress 已完成任务最多保留条数（按 UpdatedAt 淘汰最旧）。
	maxDoneProgress = 100
	// maxDoneProgressAge 已完成任务最长保留时间；与前端 DONE_KEEP 同为 1 小时量级时可再调。
	maxDoneProgressAge = 24 * time.Hour
)

// cronTask 封装单个可动态重新调度的定时任务。
type cronTask struct {
	cron  *cron.Cron
	jobID cron.EntryID
	job   func()
}

type TaskProgress struct {
	TaskID     string `json:"taskID"`
	Percentage int    `json:"percentage"`
	Message    string `json:"message"`
	Name       string `json:"name"`
	DetailMsg  string `json:"detailMsg"`
	IsDone     bool   `json:"isDone"`
	// UpdatedAt 最近一次进度变更时间（Unix 毫秒）。用于按时间淘汰与前端展示。
	UpdatedAt int64 `json:"updatedAt"`
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
		// 不 Panic：后续 API 经 RequireDocker 返回 503，/healthz 报 degraded
		logx.Errorf("Unable to create docker client: %s", err)
		cli = nil
	}
	progressPath := progressStorePath()
	progressStore := loadProgressStore(progressPath)
	nowMs := time.Now().UnixMilli()
	needsPersist := false
	for taskID, progress := range progressStore {
		changed := false
		if progress.UpdatedAt == 0 {
			progress.UpdatedAt = nowMs
			changed = true
		}
		if !progress.IsDone {
			progress.Message = "服务重启导致任务中断"
			progress.DetailMsg = "后端服务在任务完成前重启，任务未继续执行"
			progress.IsDone = true
			progress.UpdatedAt = nowMs
			changed = true
		}
		if changed {
			progressStore[taskID] = progress
			needsPersist = true
		}
	}
	ctx := &ServiceContext{
		Config:             c,
		HubImageInfo:       module.NewImageCheck(),
		ProgressStore:      progressStore,
		progressPath:       progressPath,
		ComposeTokens:      make(map[string]ComposeToken),
		composeOps:         make(map[string]string),
		DockerClient:       cli,
		updatingContainers: make(map[string]string),
	}
	before := len(ctx.ProgressStore)
	ctx.pruneProgressLocked()
	if needsPersist || len(ctx.ProgressStore) != before {
		// 启动时补时间戳/中断标记/淘汰后立即落盘，不走 debounce
		ctx.persistProgressLocked()
	}
	return ctx
}

func progressStorePath() string {
	if path := os.Getenv("TASK_PROGRESS_PATH"); path != "" {
		return path
	}
	return datadir.TaskProgressPath()
}

func loadProgressStore(path string) ProgressStoreType {
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logx.Errorf("无法读取任务进度文件 %s: %v", path, err)
		}
		return make(ProgressStoreType)
	}
	var store ProgressStoreType
	if err := json.Unmarshal(content, &store); err != nil || store == nil {
		if err != nil {
			logx.Errorf("无法解析任务进度文件 %s: %v", path, err)
		} else {
			logx.Errorf("任务进度文件 %s 不是有效对象", path)
		}
		return make(ProgressStoreType)
	}
	return store
}

// persistProgressLocked 同步写入进度文件。调用方必须已持有 ctx.mu。
func (ctx *ServiceContext) persistProgressLocked() {
	content, err := json.MarshalIndent(ctx.ProgressStore, "", "  ")
	if err != nil {
		logx.Errorf("无法序列化任务进度: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(ctx.progressPath), 0755); err != nil {
		logx.Errorf("无法创建任务进度目录: %v", err)
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(ctx.progressPath), ".taskProgress-*.tmp")
	if err != nil {
		logx.Errorf("无法创建任务进度临时文件: %v", err)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		logx.Errorf("无法设置任务进度临时文件权限: %v", err)
		return
	}
	if _, err = tmp.Write(content); err != nil {
		_ = tmp.Close()
	} else if closeErr := tmp.Close(); closeErr != nil {
		err = closeErr
	}
	if err != nil {
		logx.Errorf("无法写入任务进度: %v", err)
		return
	}
	if err := os.Rename(tmpName, ctx.progressPath); err != nil {
		logx.Errorf("无法保存任务进度: %v", err)
	}
}

// FlushProgress 取消待写定时器并立即落盘（测试或优雅退出时可用）。
func (ctx *ServiceContext) FlushProgress() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.progressPersistTimer != nil {
		ctx.progressPersistTimer.Stop()
		ctx.progressPersistTimer = nil
	}
	ctx.persistProgressLocked()
}

func (ctx *ServiceContext) scheduleProgressPersistLocked() {
	if ctx.progressPersistTimer != nil {
		return
	}
	ctx.progressPersistTimer = time.AfterFunc(progressPersistDelay, func() {
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		ctx.progressPersistTimer = nil
		ctx.persistProgressLocked()
	})
}

func (ctx *ServiceContext) UpdateProgress(taskID string, progress TaskProgress) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if progress.UpdatedAt == 0 {
		progress.UpdatedAt = time.Now().UnixMilli()
	}
	ctx.ProgressStore[taskID] = progress
	ctx.pruneProgressLocked()
	// 终态立即落盘，避免重启丢失；进行中合并写盘
	if progress.IsDone {
		if ctx.progressPersistTimer != nil {
			ctx.progressPersistTimer.Stop()
			ctx.progressPersistTimer = nil
		}
		ctx.persistProgressLocked()
		return
	}
	ctx.scheduleProgressPersistLocked()
}

// pruneProgressLocked 限制已完成任务数量与年龄，避免 taskProgress.json 无限增长。
// 按 UpdatedAt 升序淘汰最旧；无时间戳的视为最旧。调用方必须已持有 ctx.mu。
func (ctx *ServiceContext) pruneProgressLocked() {
	type doneItem struct {
		id        string
		updatedAt int64
	}
	now := time.Now().UnixMilli()
	cutoff := now - maxDoneProgressAge.Milliseconds()
	done := make([]doneItem, 0)
	for id, p := range ctx.ProgressStore {
		if !p.IsDone {
			continue
		}
		// 超龄直接删
		if p.UpdatedAt > 0 && p.UpdatedAt < cutoff {
			delete(ctx.ProgressStore, id)
			continue
		}
		done = append(done, doneItem{id: id, updatedAt: p.UpdatedAt})
	}
	if len(done) <= maxDoneProgress {
		return
	}
	sort.Slice(done, func(i, j int) bool {
		if done[i].updatedAt == done[j].updatedAt {
			return done[i].id < done[j].id
		}
		return done[i].updatedAt < done[j].updatedAt
	})
	extra := len(done) - maxDoneProgress
	for i := 0; i < extra; i++ {
		delete(ctx.ProgressStore, done[i].id)
	}
}

func (ctx *ServiceContext) GetProgress(taskID string) (TaskProgress, bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	progress, ok := ctx.ProgressStore[taskID]
	return progress, ok
}

func (ctx *ServiceContext) ListProgress() ProgressStoreType {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	store := make(ProgressStoreType, len(ctx.ProgressStore))
	for taskID, progress := range ctx.ProgressStore {
		store[taskID] = progress
	}
	return store
}

func (ctx *ServiceContext) TryStartContainerUpdate(containerID, taskID string) bool {
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	if _, exists := ctx.updatingContainers[containerID]; exists {
		return false
	}
	ctx.updatingContainers[containerID] = taskID
	return true
}

func (ctx *ServiceContext) FinishContainerUpdate(containerID, taskID string) {
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	if current, ok := ctx.updatingContainers[containerID]; ok && current == taskID {
		delete(ctx.updatingContainers, containerID)
	}
}

// TryStartComposeOp 尝试占用某 Compose 项目的操作锁（deploy/cleanup）。
// 返回 false 表示该项目已有进行中的操作。
func (ctx *ServiceContext) TryStartComposeOp(projectID, op string) bool {
	if ctx == nil || projectID == "" {
		return false
	}
	ctx.composeOpsMu.Lock()
	defer ctx.composeOpsMu.Unlock()
	if ctx.composeOps == nil {
		ctx.composeOps = make(map[string]string)
	}
	if _, exists := ctx.composeOps[projectID]; exists {
		return false
	}
	ctx.composeOps[projectID] = op
	return true
}

// FinishComposeOp 释放项目操作锁；仅当当前占用方仍是 op 时删除。
func (ctx *ServiceContext) FinishComposeOp(projectID, op string) {
	if ctx == nil || projectID == "" {
		return
	}
	ctx.composeOpsMu.Lock()
	defer ctx.composeOpsMu.Unlock()
	if current, ok := ctx.composeOps[projectID]; ok && current == op {
		delete(ctx.composeOps, projectID)
	}
}

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

// StopCrons 停止更新检查与自动备份调度器，阻塞至已触发的任务结束（cron.Stop）。
func (ctx *ServiceContext) StopCrons() {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	stopOne := func(t *cronTask) {
		if t.cron == nil {
			return
		}
		if t.jobID != 0 {
			t.cron.Remove(t.jobID)
			t.jobID = 0
		}
		// Stop 返回的 context 在运行中的 job 结束后 Done
		stopCtx := t.cron.Stop()
		<-stopCtx.Done()
		t.cron = nil
	}
	stopOne(&ctx.updateTask)
	stopOne(&ctx.backupTask)
}

// Close 优雅收尾：停 cron、刷进度、关 Docker 客户端。可重复调用。
func (ctx *ServiceContext) Close() {
	if ctx == nil {
		return
	}
	ctx.StopCrons()
	ctx.FlushProgress()
	if ctx.DockerClient != nil {
		if err := ctx.DockerClient.Close(); err != nil {
			logx.Errorf("关闭 Docker 客户端失败: %v", err)
		}
		ctx.DockerClient = nil
	}
}
