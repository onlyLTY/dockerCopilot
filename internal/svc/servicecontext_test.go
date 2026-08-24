package svc

import (
	"testing"
	"time"
)

func TestContainerUpdateLock(t *testing.T) {
	ctx := &ServiceContext{}
	if !ctx.BeginContainerUpdate("container") {
		t.Fatal("first update lock was rejected")
	}
	if ctx.BeginContainerUpdate("container") {
		t.Fatal("duplicate update lock was accepted")
	}
	ctx.EndContainerUpdate("container")
	if !ctx.BeginContainerUpdate("container") {
		t.Fatal("released update lock could not be acquired")
	}
}

func TestCleanupProgressRemovesOnlyExpiredCompletedTasks(t *testing.T) {
	ctx := &ServiceContext{ProgressStore: ProgressStoreType{
		"expired": {IsDone: true, UpdatedAt: time.Now().Add(-2 * time.Hour)},
		"recent":  {IsDone: true, UpdatedAt: time.Now()},
		"running": {IsDone: false, UpdatedAt: time.Now().Add(-2 * time.Hour)},
	}}
	if removed := ctx.CleanupProgress(time.Hour); removed != 1 {
		t.Fatalf("expected one task removed, got %d", removed)
	}
	if _, exists := ctx.ProgressStore["expired"]; exists {
		t.Fatal("expired task remains")
	}
	if _, exists := ctx.ProgressStore["recent"]; !exists {
		t.Fatal("recent task was removed")
	}
	if _, exists := ctx.ProgressStore["running"]; !exists {
		t.Fatal("running task was removed")
	}
}
