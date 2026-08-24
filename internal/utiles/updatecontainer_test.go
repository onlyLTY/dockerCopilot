package utiles

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type fakeContainerUpdateClient struct {
	events                []string
	createErr             error
	startNewErr           error
	removeOldErr          error
	cancelOnCreate        context.CancelFunc
	recoveryContextErrors []error
	pullOptions           image.PullOptions
	stopped               bool
	paused                bool
	createdNetworking     *network.NetworkingConfig
}

func (f *fakeContainerUpdateClient) ImagePull(_ context.Context, _ string, options image.PullOptions) (io.ReadCloser, error) {
	f.events = append(f.events, "pull")
	f.pullOptions = options
	return io.NopCloser(strings.NewReader("{}\n")), nil
}

func TestUpdateContainerUsesRegistryAuthentication(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"registry.example":{"auth":"dXNlcjpwYXNz"}}}`)
	client := &fakeContainerUpdateClient{}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(context.Background(), serviceContext, client, "old", "app", "registry.example/team/app:latest", false, "task"); err != nil {
		t.Fatal(err)
	}
	if client.pullOptions.RegistryAuth == "" {
		t.Fatal("private registry credentials were not passed to ImagePull")
	}
}

func (f *fakeContainerUpdateClient) ContainerInspect(context.Context, string) (container.InspectResponse, error) {
	f.events = append(f.events, "inspect")
	return container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID: "old", Name: "/app", Image: "sha256:old", State: &container.State{Running: !f.stopped, Paused: f.paused},
			HostConfig: &container.HostConfig{},
		},
		Config:          &container.Config{Image: "repo:old"},
		NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{}},
	}, nil
}

func TestUpdateContainerRejectsStaleContainerName(t *testing.T) {
	client := &fakeContainerUpdateClient{}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(context.Background(), serviceContext, client, "old", "stale-name", "repo:new", true, "task"); err == nil {
		t.Fatal("expected stale container name to be rejected")
	}
	for _, event := range client.events {
		if event == "stop-old" {
			t.Fatal("container was stopped before stale name was rejected")
		}
	}
}

func (f *fakeContainerUpdateClient) ContainerStop(context.Context, string, container.StopOptions) error {
	f.events = append(f.events, "stop-old")
	return nil
}

func (f *fakeContainerUpdateClient) ContainerPause(_ context.Context, id string) error {
	f.events = append(f.events, "pause-"+id)
	return nil
}

func (f *fakeContainerUpdateClient) ContainerUnpause(_ context.Context, id string) error {
	f.events = append(f.events, "unpause-"+id)
	return nil
}

func (f *fakeContainerUpdateClient) ContainerRename(ctx context.Context, id, name string) error {
	if name == "app" {
		f.events = append(f.events, "rename-old-back")
		f.recoveryContextErrors = append(f.recoveryContextErrors, ctx.Err())
	} else {
		f.events = append(f.events, "rename-old-away")
	}
	return nil
}

func (f *fakeContainerUpdateClient) ContainerCreate(_ context.Context, config *container.Config, _ *container.HostConfig, networking *network.NetworkingConfig, _ *ocispec.Platform, _ string) (container.CreateResponse, error) {
	f.events = append(f.events, "create-"+config.Image)
	f.createdNetworking = networking
	if f.cancelOnCreate != nil {
		f.cancelOnCreate()
	}
	if f.createErr != nil {
		return container.CreateResponse{}, f.createErr
	}
	return container.CreateResponse{ID: "new"}, nil
}

func (f *fakeContainerUpdateClient) ContainerStart(ctx context.Context, id string, _ container.StartOptions) error {
	f.events = append(f.events, "start-"+id)
	if id == "old" {
		f.recoveryContextErrors = append(f.recoveryContextErrors, ctx.Err())
	}
	if id == "new" && f.startNewErr != nil {
		return f.startNewErr
	}
	return nil
}

func (f *fakeContainerUpdateClient) ContainerRemove(_ context.Context, id string, options container.RemoveOptions) error {
	f.events = append(f.events, "remove-"+id)
	if id == "old" {
		return f.removeOldErr
	}
	if id == "new" && !options.Force {
		return errors.New("rollback removal must be forced")
	}
	return nil
}

func TestUpdateContainerRollsBackCreateFailure(t *testing.T) {
	client := &fakeContainerUpdateClient{createErr: errors.New("create failed")}
	serviceContext := testUpdateServiceContext()
	err := updateContainer(context.Background(), serviceContext, client, "old", "app", "repo:new", true, "task")
	if err == nil {
		t.Fatal("expected create failure")
	}
	assertEventOrder(t, client.events, "rename-old-away", "create-repo:new", "rename-old-back", "start-old")
	assertTaskStatus(t, serviceContext, svc.TaskStatusFailed)
}

func TestUpdateContainerRollsBackStartFailure(t *testing.T) {
	client := &fakeContainerUpdateClient{startNewErr: errors.New("start failed")}
	serviceContext := testUpdateServiceContext()
	err := updateContainer(context.Background(), serviceContext, client, "old", "app", "repo:new", true, "task")
	if err == nil {
		t.Fatal("expected start failure")
	}
	assertEventOrder(t, client.events, "start-new", "remove-new", "rename-old-back", "start-old")
	assertTaskStatus(t, serviceContext, svc.TaskStatusFailed)
}

func TestUpdateContainerUsesFreshContextForRollback(t *testing.T) {
	primaryContext, cancel := context.WithCancel(context.Background())
	client := &fakeContainerUpdateClient{
		createErr:      errors.New("create failed"),
		cancelOnCreate: cancel,
	}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(primaryContext, serviceContext, client, "old", "app", "repo:new", true, "task"); err == nil {
		t.Fatal("expected create failure")
	}
	if len(client.recoveryContextErrors) != 2 {
		t.Fatalf("expected rename and restart recovery calls, got %d", len(client.recoveryContextErrors))
	}
	for _, contextErr := range client.recoveryContextErrors {
		if contextErr != nil {
			t.Fatalf("rollback reused cancelled context: %v", contextErr)
		}
	}
}

func TestUpdateContainerKeepsSuccessfulReplacementWhenOldRemovalFails(t *testing.T) {
	client := &fakeContainerUpdateClient{removeOldErr: errors.New("busy")}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(context.Background(), serviceContext, client, "old", "app", "repo:new", true, "task"); err != nil {
		t.Fatalf("replacement succeeded but returned error: %v", err)
	}
	progress, _ := serviceContext.GetProgress("task")
	if progress.Status != svc.TaskStatusCompleted || !strings.Contains(progress.DetailMsg, "旧容器删除失败") {
		t.Fatalf("unexpected progress: %+v", progress)
	}
}

func TestUpdateContainerPreservesStoppedState(t *testing.T) {
	client := &fakeContainerUpdateClient{stopped: true}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(context.Background(), serviceContext, client, "old", "app", "repo:new", true, "task"); err != nil {
		t.Fatal(err)
	}
	for _, event := range client.events {
		if event == "stop-old" || event == "start-new" {
			t.Fatalf("stopped container state was not preserved: %v", client.events)
		}
	}
}

func TestUpdateContainerPreservesPausedState(t *testing.T) {
	client := &fakeContainerUpdateClient{paused: true}
	serviceContext := testUpdateServiceContext()
	if err := updateContainer(context.Background(), serviceContext, client, "old", "app", "repo:new", true, "task"); err != nil {
		t.Fatal(err)
	}
	assertEventOrder(t, client.events, "unpause-old", "stop-old", "start-new", "pause-new")
}

func TestNetworkingConfigForRecreateDropsOperationalFields(t *testing.T) {
	inspected := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{ID: "abcdefabcdef1234", Name: "/app"},
		Config:            &container.Config{},
		NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{
			"project_default": {
				IPAMConfig: &network.EndpointIPAMConfig{IPv4Address: "172.20.0.10"},
				Aliases:    []string{"app", "abcdefabcdef", "custom-alias"},
				NetworkID:  "runtime-network-id", EndpointID: "runtime-endpoint-id",
				IPAddress: "172.20.0.10", Gateway: "172.20.0.1", MacAddress: "02:42:ac:14:00:0a",
			},
		}},
	}
	clean := networkingConfigForRecreate(inspected).EndpointsConfig["project_default"]
	if clean == nil || clean.IPAMConfig == nil || clean.IPAMConfig.IPv4Address != "172.20.0.10" {
		t.Fatalf("configured IPAM settings were lost: %+v", clean)
	}
	if clean.NetworkID != "" || clean.EndpointID != "" || clean.IPAddress != "" || clean.Gateway != "" || clean.MacAddress != "" {
		t.Fatalf("runtime-only network fields leaked into create request: %+v", clean)
	}
	if len(clean.Aliases) != 1 || clean.Aliases[0] != "custom-alias" {
		t.Fatalf("generated aliases were not filtered: %+v", clean.Aliases)
	}
}

func testUpdateServiceContext() *svc.ServiceContext {
	return &svc.ServiceContext{
		ProgressStore: make(svc.ProgressStoreType),
		HubImageInfo:  module.NewImageCheck(),
	}
}

func assertTaskStatus(t *testing.T, serviceContext *svc.ServiceContext, expected string) {
	t.Helper()
	progress, exists := serviceContext.GetProgress("task")
	if !exists || progress.Status != expected || !progress.IsDone {
		t.Fatalf("unexpected task progress: %+v, exists=%v", progress, exists)
	}
}

func assertEventOrder(t *testing.T, events []string, expected ...string) {
	t.Helper()
	next := 0
	for _, event := range events {
		if next < len(expected) && event == expected[next] {
			next++
		}
	}
	if next != len(expected) {
		t.Fatalf("events %v do not contain ordered sequence %v", events, expected)
	}
}
