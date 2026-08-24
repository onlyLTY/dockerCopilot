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
	paused           []string
	removed          []string
	pullOptions      image.PullOptions
	createdNetwork   *network.NetworkingConfig
}

func (f *fakeRestoreClient) ImagePull(_ context.Context, _ string, options image.PullOptions) (io.ReadCloser, error) {
	f.pullOptions = options
	return io.NopCloser(strings.NewReader("{}\n")), nil
}

func TestRestoreContainerUsesRegistryAuthentication(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"registry.example":{"auth":"dXNlcjpwYXNz"}}}`)
	filename, serviceContext := writeRestoreTestBackupWithImage(t, false, "registry.example/team/app:latest")
	client := &fakeRestoreClient{}
	if err := restoreContainer(context.Background(), serviceContext, client, filename, "restore-task"); err != nil {
		t.Fatal(err)
	}
	if client.pullOptions.RegistryAuth == "" {
		t.Fatal("private registry credentials were not passed to restore ImagePull")
	}
}

func (f *fakeRestoreClient) ContainerCreate(_ context.Context, _ *container.Config, _ *container.HostConfig, networking *network.NetworkingConfig, _ *ocispec.Platform, _ string) (container.CreateResponse, error) {
	f.createdNetwork = networking
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

func (f *fakeRestoreClient) ContainerPause(_ context.Context, id string) error {
	f.paused = append(f.paused, id)
	return nil
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

func TestRestoreContainerRestoresPausedState(t *testing.T) {
	filename, serviceContext := writeRestoreTestBackupWithState(t, true, true, "repo:tag")
	client := &fakeRestoreClient{}
	if err := restoreContainer(context.Background(), serviceContext, client, filename, "restore-task"); err != nil {
		t.Fatal(err)
	}
	if len(client.started) != 1 || len(client.paused) != 1 || client.paused[0] != "restored" {
		t.Fatalf("paused state was not restored: started=%v paused=%v", client.started, client.paused)
	}
}

func TestCleanSavedNetworkingConfigDropsRuntimeFields(t *testing.T) {
	saved := &network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{
		"project_default": {
			Aliases: []string{"app", "custom"}, NetworkID: "old-network", EndpointID: "old-endpoint",
			IPAddress: "172.20.0.2", IPAMConfig: &network.EndpointIPAMConfig{IPv4Address: "172.20.0.10"},
		},
	}}
	clean := cleanSavedNetworkingConfig(saved, &container.Config{}, "app").EndpointsConfig["project_default"]
	if clean.NetworkID != "" || clean.EndpointID != "" || clean.IPAddress != "" {
		t.Fatalf("runtime fields were retained: %+v", clean)
	}
	if clean.IPAMConfig == nil || clean.IPAMConfig.IPv4Address != "172.20.0.10" || len(clean.Aliases) != 1 || clean.Aliases[0] != "custom" {
		t.Fatalf("configured network fields were lost: %+v", clean)
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
	return writeRestoreTestBackupWithImage(t, wasRunning, "repo:tag")
}

func writeRestoreTestBackupWithImage(t *testing.T, wasRunning bool, imageReference string) (string, *svc.ServiceContext) {
	return writeRestoreTestBackupWithState(t, wasRunning, false, imageReference)
}

func writeRestoreTestBackupWithState(t *testing.T, wasRunning, wasPaused bool, imageReference string) (string, *svc.ServiceContext) {
	t.Helper()
	const accessSecret = "restore-test-secret-that-is-long-enough"
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	entry := containerBackupEntry{
		ContainerCreateConfig: dockerBackend.ContainerCreateConfig{
			Name: "app", Config: &container.Config{Image: imageReference},
			HostConfig: &container.HostConfig{}, NetworkingConfig: &network.NetworkingConfig{},
		},
		WasRunning: &wasRunning,
		WasPaused:  &wasPaused,
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
