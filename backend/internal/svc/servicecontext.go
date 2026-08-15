package svc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	containerOpsMu     sync.Mutex
	containerDeletes   map[string]string
	updateSlots        chan struct{}
	imageOps           chan struct{}
	imageOpsMu         sync.Mutex
	taskMu             sync.Mutex
	taskCancels        map[string]context.CancelFunc
	shutdownCtx        context.Context
	shutdownCancel     context.CancelFunc
	closing            bool
	closeOnce          sync.Once
	taskWG             sync.WaitGroup

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
	// maxConcurrentContainerUpdates 限制实际 Docker 重建任务的并发数。
	maxConcurrentContainerUpdates = 3
	closeWaitTimeout              = 15 * time.Second
)

var ErrServiceClosing = errors.New("服务正在关闭")
var ErrActiveTask = errors.New("任务仍在执行")

// cronTask 封装单个可动态重新调度的定时任务。
type cronTask struct {
	cron  *cron.Cron
	jobID cron.EntryID
	job   func(context.Context)
	name  string
}

type TaskProgress struct {
	TaskID         string     `json:"taskID"`
	ResourceID     string     `json:"resourceID,omitempty"`
	Percentage     int        `json:"percentage"`
	Message        string     `json:"message"`
	Name           string     `json:"name"`
	DetailMsg      string     `json:"detailMsg"`
	StepPercentage int        `json:"stepPercentage"`
	ProgressType   string     `json:"progressType,omitempty"`
	Indeterminate  bool       `json:"indeterminate,omitempty"`
	Current        int64      `json:"current,omitempty"`
	Total          int64      `json:"total,omitempty"`
	IsDone         bool       `json:"isDone"`
	Failed         bool       `json:"failed"`
	Canceled       bool       `json:"canceled"`
	TimedOut       bool       `json:"timedOut"`
	Refresh        bool       `json:"refresh"`
	Steps          []TaskStep `json:"steps"`
	// UpdatedAt 最近一次进度变更时间（Unix 毫秒）。用于按时间淘汰与前端展示。
	UpdatedAt int64 `json:"updatedAt"`
}

type TaskStep struct {
	Message        string `json:"message"`
	DetailMsg      string `json:"detailMsg,omitempty"`
	StepPercentage int    `json:"stepPercentage"`
	ProgressType   string `json:"progressType,omitempty"`
	Indeterminate  bool   `json:"indeterminate,omitempty"`
	Current        int64  `json:"current,omitempty"`
	Total          int64  `json:"total,omitempty"`
	StartedAt      int64  `json:"startedAt"`
	EndedAt        int64  `json:"endedAt,omitempty"`
	DurationMs     int64  `json:"durationMs"`
	IsDone         bool   `json:"isDone"`
	Failed         bool   `json:"failed"`
}

type ProgressStoreType map[string]TaskProgress

const ProgressTypeImagePull = "image-pull"

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
			progress.Failed = true
			if len(progress.Steps) > 0 && progress.Steps[len(progress.Steps)-1].EndedAt == 0 {
				closeTaskStep(&progress.Steps[len(progress.Steps)-1], nowMs)
				progress.Steps[len(progress.Steps)-1].Failed = true
			}
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
		containerDeletes:   make(map[string]string),
		updateSlots:        make(chan struct{}, maxConcurrentContainerUpdates),
		imageOps:           make(chan struct{}, 1),
		taskCancels:        make(map[string]context.CancelFunc),
	}
	ctx.shutdownCtx, ctx.shutdownCancel = context.WithCancel(context.Background())
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

func progressFailed(progress TaskProgress) bool {
	if !progress.IsDone || progress.Canceled {
		return false
	}
	if progress.Failed || progress.Percentage < 100 {
		return true
	}
	text := strings.ToLower(progress.Message + " " + progress.DetailMsg)
	for _, keyword := range []string{"失败", "错误", "中断", "拒绝", "超时", "error", "fail", "timeout", "interrupted"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func stepFailed(progress TaskProgress) bool {
	if progress.Failed {
		return true
	}
	text := strings.ToLower(progress.Message + " " + progress.DetailMsg)
	for _, keyword := range []string{"失败", "错误", "中断", "拒绝", "超时", "error", "fail", "timeout", "interrupted"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (ctx *ServiceContext) UpdateProgress(taskID string, progress TaskProgress) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	previous, exists := ctx.ProgressStore[taskID]
	if exists && (previous.IsDone || previous.Canceled) {
		return
	}
	if exists && previous.UpdatedAt > 0 && progress.UpdatedAt > 0 && progress.UpdatedAt < previous.UpdatedAt {
		return
	}
	ctx.updateProgressLocked(taskID, progress, previous)
}

func (ctx *ServiceContext) updateProgressLocked(taskID string, progress, previous TaskProgress) {
	now := time.Now().UnixMilli()
	if progress.UpdatedAt == 0 || progress.UpdatedAt < previous.UpdatedAt {
		progress.UpdatedAt = now
	}
	if progress.ResourceID == "" {
		progress.ResourceID = previous.ResourceID
	}
	if !progress.Refresh {
		progress.Refresh = previous.Refresh
	}
	if progress.ProgressType != ProgressTypeImagePull {
		progress.ProgressType = ""
		progress.Indeterminate = false
		progress.Current = 0
		progress.Total = 0
	}
	if progress.IsDone {
		progress.Failed = progressFailed(progress)
	}
	progress.Steps = append([]TaskStep(nil), previous.Steps...)
	if progress.Message != "" {
		failed := stepFailed(progress)
		if len(progress.Steps) == 0 || progress.Steps[len(progress.Steps)-1].Message != progress.Message {
			if len(progress.Steps) > 0 && progress.Steps[len(progress.Steps)-1].EndedAt == 0 {
				closeTaskStep(&progress.Steps[len(progress.Steps)-1], now)
			}
			progress.Steps = append(progress.Steps, TaskStep{
				Message:        progress.Message,
				DetailMsg:      progress.DetailMsg,
				StepPercentage: progress.StepPercentage,
				ProgressType:   progress.ProgressType,
				Indeterminate:  progress.Indeterminate,
				Current:        progress.Current,
				Total:          progress.Total,
				StartedAt:      now,
				IsDone:         progress.IsDone,
				Failed:         failed,
			})
		} else {
			step := &progress.Steps[len(progress.Steps)-1]
			step.DetailMsg = progress.DetailMsg
			step.StepPercentage = progress.StepPercentage
			step.ProgressType = progress.ProgressType
			step.Indeterminate = progress.Indeterminate
			step.Current = progress.Current
			step.Total = progress.Total
			step.IsDone = progress.IsDone
			step.Failed = failed
		}
	}
	if progress.IsDone && len(progress.Steps) > 0 && progress.Steps[len(progress.Steps)-1].EndedAt == 0 {
		closeTaskStep(&progress.Steps[len(progress.Steps)-1], now)
	}
	ctx.ProgressStore[taskID] = progress
	ctx.pruneProgressLocked()
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

func closeTaskStep(step *TaskStep, endedAt int64) {
	step.EndedAt = endedAt
	step.DurationMs = endedAt - step.StartedAt
	step.IsDone = true
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

// DeleteProgress 删除指定任务进度并立即落盘。活动任务必须先取消并等待终态。
func (ctx *ServiceContext) DeleteProgress(taskID string) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if progress, exists := ctx.ProgressStore[taskID]; exists && !progress.IsDone {
		return ErrActiveTask
	}
	delete(ctx.ProgressStore, taskID)
	ctx.persistProgressLocked()
	return nil
}

// ClearProgress 清空任务进度记录，doneOnly=true 时只清空已完成任务。
// 为避免活动 worker 丢失控制入口，doneOnly=false 也只清理已完成记录并返回活动数量。
func (ctx *ServiceContext) ClearProgress(doneOnly bool) int {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	active := 0
	for id, p := range ctx.ProgressStore {
		if !p.IsDone {
			active++
			continue
		}
		if p.IsDone {
			delete(ctx.ProgressStore, id)
		}
	}
	ctx.persistProgressLocked()
	return active
}

// RegisterTask registers a cancellable worker. The returned context is also
// canceled when the service starts shutting down.
func (ctx *ServiceContext) RegisterTask(taskID string) context.Context {
	ctx.taskMu.Lock()
	defer ctx.taskMu.Unlock()
	if ctx.closing {
		return nil
	}
	if ctx.taskCancels == nil {
		ctx.taskCancels = make(map[string]context.CancelFunc)
	}
	if _, exists := ctx.taskCancels[taskID]; exists {
		return nil
	}
	base := ctx.shutdownCtx
	if base == nil {
		base = context.Background()
	}
	taskCtx, cancel := context.WithCancel(base)
	ctx.taskCancels[taskID] = cancel
	ctx.taskWG.Add(1)
	return taskCtx
}

// FinishTask releases the cancellation controller after the worker reaches a terminal state.
func (ctx *ServiceContext) FinishTask(taskID string) {
	ctx.taskMu.Lock()
	if _, exists := ctx.taskCancels[taskID]; exists {
		delete(ctx.taskCancels, taskID)
		ctx.taskWG.Done()
	}
	ctx.taskMu.Unlock()
}

// CancelTask requests cancellation and marks the progress as terminal only when the worker observes it.
func (ctx *ServiceContext) CancelTask(taskID string) (TaskProgress, bool, bool) {
	progress, exists := ctx.GetProgress(taskID)
	if !exists {
		return TaskProgress{}, false, false
	}
	if progress.IsDone {
		return progress, true, false
	}
	ctx.taskMu.Lock()
	cancel, active := ctx.taskCancels[taskID]
	ctx.taskMu.Unlock()
	if active {
		cancel()
	}
	return progress, true, active
}

func (ctx *ServiceContext) MarkTaskCanceled(taskID string, detail string) bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	progress, ok := ctx.ProgressStore[taskID]
	if !ok || progress.IsDone {
		return false
	}
	progress.Canceled = true
	progress.Failed = false
	progress.IsDone = true
	progress.Percentage = 100
	progress.Message = "任务已取消"
	if detail == "" {
		detail = "用户请求停止任务；已执行的 Docker 操作不会自动回滚"
	}
	progress.DetailMsg = detail
	ctx.updateProgressLocked(taskID, progress, ctx.ProgressStore[taskID])
	return true
}
func (ctx *ServiceContext) TryStartContainerUpdate(containerID, taskID string) bool {
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	ctx.containerOpsMu.Lock()
	defer ctx.containerOpsMu.Unlock()
	if ctx.updatingContainers == nil {
		ctx.updatingContainers = make(map[string]string)
	}
	if ctx.containerDeletes != nil {
		if _, exists := ctx.containerDeletes[containerID]; exists {
			return false
		}
	}
	if _, exists := ctx.updatingContainers[containerID]; exists {
		return false
	}
	ctx.updatingContainers[containerID] = taskID
	return true
}

func (ctx *ServiceContext) AcquireContainerUpdateSlot(taskContexts ...context.Context) bool {
	taskCtx := context.Background()
	if len(taskContexts) > 0 && taskContexts[0] != nil {
		taskCtx = taskContexts[0]
	}
	if ctx.updateSlots == nil {
		ctx.updateMu.Lock()
		if ctx.updateSlots == nil {
			ctx.updateSlots = make(chan struct{}, maxConcurrentContainerUpdates)
		}
		ctx.updateMu.Unlock()
	}
	select {
	case ctx.updateSlots <- struct{}{}:
		return true
	case <-taskCtx.Done():
		return false
	}
}

func (ctx *ServiceContext) ReleaseContainerUpdateSlot() {
	if ctx == nil || ctx.updateSlots == nil {
		return
	}
	select {
	case <-ctx.updateSlots:
	default:
	}
}

func (ctx *ServiceContext) FinishContainerUpdate(containerID, taskID string) {
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	ctx.containerOpsMu.Lock()
	defer ctx.containerOpsMu.Unlock()
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

func (ctx *ServiceContext) AcquireImageOp(taskCtx context.Context) bool {
	if ctx == nil {
		return false
	}
	ctx.imageOpsMu.Lock()
	if ctx.imageOps == nil {
		ctx.imageOps = make(chan struct{}, 1)
	}
	slots := ctx.imageOps
	ctx.imageOpsMu.Unlock()
	if taskCtx == nil {
		taskCtx = context.Background()
	}
	select {
	case slots <- struct{}{}:
		return true
	case <-taskCtx.Done():
		return false
	}
}

func (ctx *ServiceContext) ReleaseImageOp() {
	if ctx == nil {
		return
	}
	ctx.imageOpsMu.Lock()
	slots := ctx.imageOps
	ctx.imageOpsMu.Unlock()
	if slots == nil {
		return
	}
	select {
	case <-slots:
	default:
	}
}

func (ctx *ServiceContext) StartUpdateCron(spec string, job func(context.Context)) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.startLocked(&ctx.updateTask, "update", spec, job)
}

// RescheduleUpdateCron 按新的 cron 表达式重新调度更新检查任务。
func (ctx *ServiceContext) RescheduleUpdateCron(spec string) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.rescheduleLocked(&ctx.updateTask, spec)
}

// StartBackupCron 启动自动备份定时任务；job 为要执行的备份逻辑（由 main 注入）。
func (ctx *ServiceContext) StartBackupCron(spec string, job func(context.Context)) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.startLocked(&ctx.backupTask, "backup", spec, job)
}

// RescheduleBackupCron 按新的 cron 表达式重新调度自动备份任务。
func (ctx *ServiceContext) RescheduleBackupCron(spec string) error {
	ctx.cronMu.Lock()
	defer ctx.cronMu.Unlock()
	return ctx.rescheduleLocked(&ctx.backupTask, spec)
}

// startLocked 首次初始化调度器并注入 job，然后按 spec 调度。需持有 cronMu。
func (ctx *ServiceContext) startLocked(t *cronTask, name, spec string, job func(context.Context)) error {
	t.job = job
	t.name = name
	if t.cron == nil {
		t.cron = cron.New(
			cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
			cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow)),
		)
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

// scheduleLocked 先添加新任务，成功后再移除旧任务；spec 为空表示关闭。
func (ctx *ServiceContext) scheduleLocked(t *cronTask, spec string) error {
	var next cron.EntryID
	if spec != "" {
		var err error
		next, err = t.cron.AddFunc(spec, func() {
			taskID := "cron-" + t.name + "-" + time.Now().Format("20060102150405.000000000")
			taskCtx := ctx.RegisterTask(taskID)
			if taskCtx == nil {
				return
			}
			defer ctx.FinishTask(taskID)
			t.job(taskCtx)
		})
		if err != nil {
			return err
		}
	}
	if t.jobID != 0 {
		t.cron.Remove(t.jobID)
	}
	t.jobID = next
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

// Close 优雅收尾：停止新任务、取消并等待 worker、刷进度、关 Docker 客户端。可重复调用。
func (ctx *ServiceContext) Close() {
	if ctx == nil {
		return
	}
	ctx.closeOnce.Do(func() {
		ctx.taskMu.Lock()
		ctx.closing = true
		if ctx.shutdownCancel != nil {
			ctx.shutdownCancel()
		}
		for _, cancel := range ctx.taskCancels {
			cancel()
		}
		ctx.taskMu.Unlock()

		ctx.StopCrons()
		done := make(chan struct{})
		go func() {
			ctx.taskWG.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(closeWaitTimeout):
			logx.Errorf("等待后台任务退出超时")
		}
		ctx.FlushProgress()
		if ctx.DockerClient != nil {
			if err := ctx.DockerClient.Close(); err != nil {
				logx.Errorf("关闭 Docker 客户端失败: %v", err)
			}
			ctx.DockerClient = nil
		}
	})
}
