package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles/compose_project"
)

func TestRunComposeProjectBackupBatchContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	projectRoot := filepath.Join(root, "project")
	if err := os.MkdirAll(projectRoot, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "compose.yaml"), []byte("name: good\nservices:\n  app:\n    image: example/app\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := &svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{
		ScanPaths: []string{root},
		BackupDir: backupRoot,
	}}, ProgressStore: make(svc.ProgressStoreType)}
	taskCtx := ctx.RegisterTask("batch-test")
	runComposeProjectBackupBatch(taskCtx, ctx, "batch-test", []string{"missing", compose_project.ProjectID(projectRoot)})

	progress, ok := ctx.GetProgress("batch-test")
	if !ok || !progress.IsDone {
		t.Fatalf("expected completed progress, got %#v", progress)
	}
	if !progress.Failed || len(progress.Steps) < 3 {
		t.Fatalf("expected failed aggregate with per-project steps, got %#v", progress)
	}
	if progress.Steps[0].Failed == false || progress.Steps[1].Failed {
		t.Fatalf("unexpected step failures: %#v", progress.Steps)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, "good_compose.yaml")); err != nil {
		t.Fatalf("successful project was not backed up: %v", err)
	}
}

func TestRunComposeProjectBackupBatchCanBeCanceled(t *testing.T) {
	ctx := &svc.ServiceContext{ProgressStore: make(svc.ProgressStoreType)}
	taskCtx := ctx.RegisterTask("batch-cancel")
	ctx.UpdateProgress("batch-cancel", svc.TaskProgress{TaskID: "batch-cancel", Name: composeProjectBackupTaskName, Message: "任务已提交"})
	if _, ok, active := ctx.CancelTask("batch-cancel"); !ok || !active {
		t.Fatal("expected active task cancellation")
	}
	runComposeProjectBackupBatch(taskCtx, ctx, "batch-cancel", []string{"missing"})
	progress, ok := ctx.GetProgress("batch-cancel")
	if !ok || !progress.IsDone || !progress.Canceled {
		t.Fatalf("expected canceled progress, got %#v", progress)
	}
}
