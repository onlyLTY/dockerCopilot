package svc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/config"
)

func TestProgressStorePersistsAndRestores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	t.Setenv("TASK_PROGRESS_PATH", path)

	ctx := &ServiceContext{ProgressStore: make(ProgressStoreType), progressPath: path}
	ctx.UpdateProgress("done", TaskProgress{TaskID: "done", Name: "完成", Percentage: 100, Message: "完成", IsDone: true})
	ctx.UpdateProgress("active", TaskProgress{TaskID: "active", Name: "进行中", Percentage: 40, Message: "执行中"})

	stored := loadProgressStore(path)
	if len(stored) != 2 || stored["active"].Percentage != 40 {
		t.Fatalf("unexpected stored progress: %+v", stored)
	}
}

func TestNewServiceContextMarksInterruptedTasks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	t.Setenv("TASK_PROGRESS_PATH", path)
	initial := ProgressStoreType{
		"active": {TaskID: "active", Name: "部署项目", Percentage: 50, Message: "部署中"},
		"done":   {TaskID: "done", Name: "已完成", Percentage: 100, Message: "完成", IsDone: true},
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
			ctx.UpdateProgress(id, TaskProgress{TaskID: id, Percentage: i})
		}(i)
	}
	wg.Wait()
	if len(ctx.ProgressStore) != 8 {
		t.Fatalf("expected 8 tasks, got %d", len(ctx.ProgressStore))
	}
}
