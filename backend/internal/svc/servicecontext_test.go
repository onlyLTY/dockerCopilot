package svc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/config"
)

func TestProgressStorePersistsAndRestores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	t.Setenv("TASK_PROGRESS_PATH", path)

	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("done", TaskProgress{TaskID: "done", Name: "完成", Percentage: 100, Message: "完成", IsDone: true})
	ctx.UpdateProgress("active", TaskProgress{TaskID: "active", Name: "进行中", Percentage: 40, Message: "执行中"})
	ctx.FlushProgress()

	stored := loadProgressStore(path)
	if len(stored) != 2 || stored["active"].Percentage != 40 {
		t.Fatalf("unexpected stored progress: %+v", stored)
	}
	if stored["done"].UpdatedAt == 0 || stored["active"].UpdatedAt == 0 {
		t.Fatalf("expected UpdatedAt to be set: %+v", stored)
	}
}

func TestNewServiceContextMarksInterruptedTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	t.Setenv("TASK_PROGRESS_PATH", path)
	initial := ProgressStoreType{
		"active": {TaskID: "active", Name: "部署项目", Percentage: 50, Message: "部署中"},
		"done":   {TaskID: "done", Name: "已完成", Percentage: 100, Message: "完成", IsDone: true, UpdatedAt: time.Now().UnixMilli()},
	}
	content, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	ctx := NewServiceContext(config.Config{})
	active, ok := ctx.GetProgress("active")
	if !ok || !active.IsDone || active.Message != "服务重启导致任务中断" {
		t.Fatalf("active task was not interrupted: %+v, exists=%t", active, ok)
	}
	if active.UpdatedAt == 0 {
		t.Fatal("interrupted task should have UpdatedAt")
	}
	done, ok := ctx.GetProgress("done")
	if !ok || !done.IsDone || done.Message != "完成" {
		t.Fatalf("completed task changed unexpectedly: %+v, exists=%t", done, ok)
	}
}

func TestProgressFailureClosesFailedStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("task", TaskProgress{TaskID: "task", Percentage: 60, Message: "正在启动容器"})
	ctx.UpdateProgress("task", TaskProgress{
		TaskID: "task", Percentage: 90, Message: "启动容器失败（web）",
		DetailMsg: "daemon refused the request", Failed: true, IsDone: true,
	})
	progress, ok := ctx.GetProgress("task")
	if !ok || !progress.Failed || len(progress.Steps) != 2 {
		t.Fatalf("expected failed task with two steps: %+v", progress)
	}
	last := progress.Steps[len(progress.Steps)-1]
	if !last.IsDone || !last.Failed || last.EndedAt == 0 || last.DurationMs < 0 {
		t.Fatalf("failed step was not closed: %+v", last)
	}
}

func TestNewServiceContextClosesInterruptedStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	t.Setenv("TASK_PROGRESS_PATH", path)
	initial := ProgressStoreType{
		"active": {
			TaskID: "active", Percentage: 40, Message: "正在下载镜像",
			Steps: []TaskStep{{Message: "正在下载镜像", StartedAt: time.Now().UnixMilli() - 1000}},
		},
	}
	content, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	ctx := NewServiceContext(config.Config{})
	progress, ok := ctx.GetProgress("active")
	if !ok || !progress.Failed || len(progress.Steps) != 1 {
		t.Fatalf("expected interrupted task failure: %+v", progress)
	}
	step := progress.Steps[0]
	if !step.IsDone || !step.Failed || step.EndedAt == 0 || step.DurationMs <= 0 {
		t.Fatalf("interrupted step was not closed: %+v", step)
	}
}

func TestProgressStoreConcurrentUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i))
			ctx.UpdateProgress(id, TaskProgress{TaskID: id, Percentage: i, IsDone: true})
		}(i)
	}
	wg.Wait()
	if len(ctx.ProgressStore) != 8 {
		t.Fatalf("expected 8 tasks, got %d", len(ctx.ProgressStore))
	}
}

func TestPruneProgressByUpdatedAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	now := time.Now().UnixMilli()
	// 插入超过 maxDoneProgress 的已完成任务，旧的应被淘汰
	for i := 0; i < maxDoneProgress+5; i++ {
		id := "done-" + strconv.Itoa(i)
		ctx.ProgressStore[id] = TaskProgress{
			TaskID:    id,
			IsDone:    true,
			UpdatedAt: now - int64((maxDoneProgress+5-i)*1000),
		}
	}
	// 再塞一个进行中的，不应被 prune
	ctx.ProgressStore["running"] = TaskProgress{TaskID: "running", IsDone: false, UpdatedAt: now}

	ctx.mu.Lock()
	ctx.pruneProgressLocked()
	ctx.mu.Unlock()

	if len(ctx.ProgressStore) > maxDoneProgress+1 {
		t.Fatalf("expected at most %d entries, got %d", maxDoneProgress+1, len(ctx.ProgressStore))
	}
	if _, ok := ctx.ProgressStore["running"]; !ok {
		t.Fatal("running task should not be pruned")
	}
	// 最旧的应被删掉
	if _, ok := ctx.ProgressStore["done-0"]; ok {
		t.Fatal("oldest done task should be pruned")
	}
}

func TestProgressStepPercentageKeepsZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("pull", TaskProgress{TaskID: "pull", Message: "正在拉取新镜像", ProgressType: ProgressTypeImagePull, Current: 25, Total: 100, StepPercentage: 0})
	progress, ok := ctx.GetProgress("pull")
	if !ok || len(progress.Steps) != 1 || progress.Steps[0].StepPercentage != 0 {
		t.Fatalf("expected zero step percentage to be retained: %+v", progress)
	}
	if progress.Steps[0].ProgressType != ProgressTypeImagePull || progress.Steps[0].Current != 25 || progress.Steps[0].Total != 100 {
		t.Fatalf("expected pull metadata to be retained: %+v", progress.Steps[0])
	}
}

func TestProgressStepWithoutPullTypeHasNoVisualProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("task", TaskProgress{TaskID: "task", Message: "正在停止容器", StepPercentage: 0})
	progress, ok := ctx.GetProgress("task")
	if !ok || len(progress.Steps) != 1 || progress.Steps[0].ProgressType != "" {
		t.Fatalf("expected regular step without visual progress type: %+v", progress)
	}
}

func TestProgressPullUpdatesSingleStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("pull", TaskProgress{TaskID: "pull", Message: "正在拉取新镜像", ProgressType: ProgressTypeImagePull, Current: 10, Total: 100, StepPercentage: 10})
	ctx.UpdateProgress("pull", TaskProgress{TaskID: "pull", Message: "正在拉取新镜像", ProgressType: ProgressTypeImagePull, Current: 80, Total: 100, StepPercentage: 80})
	progress, ok := ctx.GetProgress("pull")
	if !ok || len(progress.Steps) != 1 || progress.Steps[0].Current != 80 || progress.Steps[0].StepPercentage != 80 {
		t.Fatalf("expected pull updates to replace one step: %+v", progress)
	}
}

func TestProgressStagesOnlyPullUsesOneStep(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	stages := []TaskProgress{
		{TaskID: "pull", Message: "正在连接镜像源"},
		{TaskID: "pull", Message: "等待镜像层"},
		{TaskID: "pull", Message: "正在拉取镜像", ProgressType: ProgressTypeImagePull, Current: 40, Total: 100, StepPercentage: 40},
		{TaskID: "pull", Message: "正在拉取镜像", DetailMsg: "正在解压镜像", ProgressType: ProgressTypeImagePull, Indeterminate: true, StepPercentage: 100},
	}
	for _, stage := range stages {
		ctx.UpdateProgress("pull", stage)
	}
	progress, ok := ctx.GetProgress("pull")
	if !ok || len(progress.Steps) != 3 {
		t.Fatalf("expected three stages with one pull step: %+v", progress)
	}
	pullStep := progress.Steps[2]
	if pullStep.Message != "正在拉取镜像" || !pullStep.Indeterminate || pullStep.StepPercentage != 100 {
		t.Fatalf("pull step was not updated for extraction: %+v", pullStep)
	}
}

func TestProgressStagesOnlyDownloadHasProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	stages := []TaskProgress{
		{TaskID: "pull", Message: "正在连接镜像源"},
		{TaskID: "pull", Message: "等待镜像层"},
		{TaskID: "pull", Message: "正在拉取镜像", ProgressType: ProgressTypeImagePull, Current: 40, Total: 100, StepPercentage: 40},
		{TaskID: "pull", Message: "正在拉取镜像", DetailMsg: "正在解压镜像", ProgressType: ProgressTypeImagePull, Indeterminate: true, StepPercentage: 100},
	}
	for _, stage := range stages {
		ctx.UpdateProgress("pull", stage)
	}
	progress, ok := ctx.GetProgress("pull")
	if !ok || len(progress.Steps) != 3 {
		t.Fatalf("expected three pull stages: %+v", progress)
	}
	if progress.Steps[2].ProgressType != ProgressTypeImagePull || progress.Steps[2].Current != 0 || progress.Steps[2].Total != 0 || !progress.Steps[2].Indeterminate {
		t.Fatalf("pull stage should represent extraction processing: %+v", progress.Steps[2])
	}
	if progress.Steps[2].DetailMsg != "正在解压镜像" {
		t.Fatalf("pull stage should retain extraction detail: %+v", progress.Steps[2])
	}
}
func TestProgressDebounceFlushesOnDone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("a", TaskProgress{TaskID: "a", Percentage: 10, Message: "进行中"})
	ctx.UpdateProgress("a", TaskProgress{TaskID: "a", Percentage: 100, Message: "完成", IsDone: true})
	stored := loadProgressStore(path)
	if !stored["a"].IsDone || stored["a"].Percentage != 100 {
		t.Fatalf("done progress should be flushed immediately: %+v", stored["a"])
	}
}

func TestProgressResourceIDIsInherited(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("task", TaskProgress{TaskID: "task", ResourceID: "container-1", Message: "开始"})
	ctx.UpdateProgress("task", TaskProgress{TaskID: "task", Message: "执行中"})
	progress, ok := ctx.GetProgress("task")
	if !ok || progress.ResourceID != "container-1" {
		t.Fatalf("resource ID was not inherited: %+v", progress)
	}
}

func TestContainerUpdateSlotLimitsAndReleases(t *testing.T) {
	ctx := &ServiceContext{updateSlots: make(chan struct{}, 2)}
	ctx.AcquireContainerUpdateSlot()
	ctx.AcquireContainerUpdateSlot()
	acquired := make(chan struct{})
	go func() {
		ctx.AcquireContainerUpdateSlot()
		close(acquired)
	}()
	select {
	case <-acquired:
		t.Fatal("third update acquired a full slot set")
	case <-time.After(20 * time.Millisecond):
	}
	ctx.ReleaseContainerUpdateSlot()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("waiting update did not acquire a released slot")
	}
	ctx.ReleaseContainerUpdateSlot()
	ctx.ReleaseContainerUpdateSlot()
}

func TestContainerUpdateLockIsExclusive(t *testing.T) {
	ctx := &ServiceContext{}
	if !ctx.TryStartContainerUpdate("container-1", "task-1") {
		t.Fatal("first update was rejected")
	}
	if ctx.TryStartContainerUpdate("container-1", "task-2") {
		t.Fatal("second update for the same container was accepted")
	}
	ctx.FinishContainerUpdate("container-1", "task-1")
	if !ctx.TryStartContainerUpdate("container-1", "task-2") {
		t.Fatal("container lock was not released")
	}
}
