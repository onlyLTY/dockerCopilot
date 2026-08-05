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

func TestProgressDebounceFlushesOnDone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("a", TaskProgress{TaskID: "a", Percentage: 10, Message: "进行中"})
	// 未 Flush 前文件可能尚不完整；完成后应立即可见
	ctx.UpdateProgress("a", TaskProgress{TaskID: "a", Percentage: 100, Message: "完成", IsDone: true})
	stored := loadProgressStore(path)
	if !stored["a"].IsDone || stored["a"].Percentage != 100 {
		t.Fatalf("done progress should be flushed immediately: %+v", stored["a"])
	}
}
