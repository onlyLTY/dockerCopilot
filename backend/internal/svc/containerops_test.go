package svc

import "testing"

func TestContainerDeleteLockConflictsWithUpdate(t *testing.T) {
	ctx := &ServiceContext{}
	if !ctx.TryStartContainerDelete("container-1", "delete-1") {
		t.Fatal("first delete was rejected")
	}
	if ctx.TryStartContainerUpdate("container-1", "update-1") {
		t.Fatal("update started while delete was active")
	}
	ctx.FinishContainerDelete("container-1", "delete-1")
	if !ctx.TryStartContainerUpdate("container-1", "update-1") {
		t.Fatal("update did not start after delete released")
	}
	ctx.FinishContainerUpdate("container-1", "update-1")
}

func TestContainerUpdateLockConflictsWithDelete(t *testing.T) {
	ctx := &ServiceContext{}
	if !ctx.TryStartContainerUpdate("container-1", "update-1") {
		t.Fatal("first update was rejected")
	}
	if ctx.TryStartContainerDelete("container-1", "delete-1") {
		t.Fatal("delete started while update was active")
	}
	ctx.FinishContainerUpdate("container-1", "update-1")
	if !ctx.TryStartContainerDelete("container-1", "delete-1") {
		t.Fatal("delete did not start after update released")
	}
	ctx.FinishContainerDelete("container-1", "delete-1")
}
