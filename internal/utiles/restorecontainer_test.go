package utiles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dockerBackend "github.com/docker/docker/api/types/backend"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type fakeRestoreClient struct {
	createErr        error
	startErr         error
	cancelOnStart    context.CancelFunc
	removeContextErr error
	started          []string
	removed          []string
}

func (f *fakeRestoreClient) ImagePull(context.Context, string, image.PullOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("{}\n")), nil
}

func (f *fakeRestoreClient) ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, *ocispec.Platform, string) (container.CreateResponse, error) {
	if f.createErr != nil {
		return container.CreateResponse{}, f.createErr
	}
	return container.CreateResponse{ID: "restored"}, nil
}

func (f *fakeRestoreClient) ContainerStart(_ context.Context, id string, _ container.StartOptions) error {
	f.started = append(f.started, id)
	if f.cancelOnStart != nil {
		f.cancelOnStart()
	}
	return f.startErr
}

func (f *fakeRestoreClient) ContainerRemove(ctx context.Context, id string, _ container.RemoveOptions) error {
	f.removed = append(f.removed, id)
	f.removeContextErr = ctx.Err()
	return nil
}

func TestRestoreContainerReportsPartialFailure(t *testing.T) {
	filename, serviceContext := writeRestoreTestBackup(t, true)
	client := &fakeRestoreClient{createErr: errors.New("name conflict")}
	err := restoreContainer(context.Background(), serviceContext, client, filename, "restore-task")
	if err == nil {
		t.Fatal("expected restore failure")
	}
	progress, _ := serviceContext.GetProgress("restore-task")
	if progress.Status != svc.TaskStatusFailed || !progress.IsDone || !strings.Contains(progress.DetailMsg, "name conflict") {
		t.Fatalf("partial failure was not reported: %+v", progress)
	}
}

func TestRestoreContainerRestoresRunningState(t *testing.T) {
	filename, serviceContext := writeRestoreTestBackup(t, true)
	client := &fakeRestoreClient{}
	if err := restoreContainer(context.Background(), serviceContext, client, filename, "restore-task"); err != nil {
		t.Fatal(err)
	}
	if len(client.started) != 1 || client.started[0] != "restored" {
		t.Fatalf("running container was not restarted: %v", client.started)
	}
	progress, _ := serviceContext.GetProgress("restore-task")
	if progress.Status != svc.TaskStatusCompleted || !progress.IsDone {
		t.Fatalf("restore did not complete: %+v", progress)
	}
}

func TestRestoreContainerUsesFreshContextForCleanup(t *testing.T) {
	filename, serviceContext := writeRestoreTestBackup(t, true)
	primaryContext, cancel := context.WithCancel(context.Background())
	client := &fakeRestoreClient{startErr: errors.New("start failed"), cancelOnStart: cancel}
	if err := restoreContainer(primaryContext, serviceContext, client, filename, "restore-task"); err == nil {
		t.Fatal("expected restore failure")
	}
	if len(client.removed) != 1 || client.removed[0] != "restored" {
		t.Fatalf("failed container was not removed: %v", client.removed)
	}
	if client.removeContextErr != nil {
		t.Fatalf("cleanup reused cancelled context: %v", client.removeContextErr)
	}
}

func writeRestoreTestBackup(t *testing.T, wasRunning bool) (string, *svc.ServiceContext) {
	t.Helper()
	const accessSecret = "restore-test-secret-that-is-long-enough"
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	entry := containerBackupEntry{
		ContainerCreateConfig: dockerBackend.ContainerCreateConfig{
			Name: "app", Config: &container.Config{Image: "repo:tag"},
			HostConfig: &container.HostConfig{}, NetworkingConfig: &network.NetworkingConfig{},
		},
		WasRunning: &wasRunning,
	}
	plaintext, err := json.Marshal([]containerBackupEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := encryptBackup(plaintext, accessSecret)
	if err != nil {
		t.Fatal(err)
	}
	filename := "backup-test.json"
	if err := os.WriteFile(filepath.Join(dir, filename), encrypted, 0o600); err != nil {
		t.Fatal(err)
	}
	return filename, &svc.ServiceContext{
		Config: config.Config{Auth: struct {
			AccessSecret string
			AccessExpire int64
		}{AccessSecret: accessSecret}},
		ProgressStore: make(svc.ProgressStoreType),
	}
}
