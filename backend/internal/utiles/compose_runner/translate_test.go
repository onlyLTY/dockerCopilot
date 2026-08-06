package compose_runner

import (
	"testing"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
)

func strPtr(s string) *string { return &s }

func TestEnvironmentSlice(t *testing.T) {
	env := composeTypes.MappingWithEquals{
		"B_KEY": strPtr("2"),
		"A_KEY": strPtr("1"),
		"NOVAL": nil,
	}
	got := environmentSlice(env)
		// 按 key 排序；nil 值透传为裸 KEY
		want := []string{"A_KEY=1", "B_KEY=2", "NOVAL"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
	if environmentSlice(nil) != nil {
		t.Error("nil environment should return nil")
	}
}

func TestTranslateRestart(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantMax  int
	}{
		{"", "", 0},
		{"always", "always", 0},
		{"unless-stopped", "unless-stopped", 0},
		{"on-failure", "on-failure", 0},
		{"on-failure:5", "on-failure", 5},
	}
	for _, c := range cases {
		got := translateRestart(c.in)
		if string(got.Name) != c.wantName || got.MaximumRetryCount != c.wantMax {
			t.Errorf("translateRestart(%q) = {%s,%d}, want {%s,%d}", c.in, got.Name, got.MaximumRetryCount, c.wantName, c.wantMax)
		}
	}
}

func TestTranslatePorts(t *testing.T) {
	ports := []composeTypes.ServicePortConfig{
		{Target: 8080, Published: "80", Protocol: "tcp"},
		{Target: 53, Protocol: "udp"}, // exposed only, no binding
	}
	exposed, bindings, err := translatePorts(ports)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exposed) != 2 {
		t.Errorf("exposed count = %d, want 2", len(exposed))
	}
	if len(bindings) != 1 {
		t.Errorf("bindings count = %d, want 1", len(bindings))
	}
	for p, b := range bindings {
		if p.Port() != "8080" || p.Proto() != "tcp" {
			t.Errorf("binding port = %s/%s, want 8080/tcp", p.Port(), p.Proto())
		}
		if len(b) != 1 || b[0].HostPort != "80" {
			t.Errorf("host binding = %+v, want HostPort 80", b)
		}
	}
}

func TestTranslatePortsDefaultProto(t *testing.T) {
	exposed, _, err := translatePorts([]composeTypes.ServicePortConfig{{Target: 3000}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for p := range exposed {
		if p.Proto() != "tcp" {
			t.Errorf("default proto = %s, want tcp", p.Proto())
		}
	}
}

func TestTranslateVolumes(t *testing.T) {
	vols := []composeTypes.ServiceVolumeConfig{
		{Type: "bind", Source: "/host/data", Target: "/data"},
		{Type: "bind", Source: "/host/ro", Target: "/ro", ReadOnly: true},
		{Type: "volume", Source: "mydata", Target: "/var/lib"},
		{Type: "tmpfs", Target: "/tmp"},
	}
	volNames := map[string]string{"mydata": "proj_mydata"}
	binds, mounts, err := translateVolumes(vols, volNames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantBinds := []string{"/host/data:/data", "/host/ro:/ro:ro"}
	if len(binds) != len(wantBinds) {
		t.Fatalf("binds = %v, want %v", binds, wantBinds)
	}
	for i := range wantBinds {
		if binds[i] != wantBinds[i] {
			t.Errorf("bind %d = %q, want %q", i, binds[i], wantBinds[i])
		}
	}
	if len(mounts) != 2 {
		t.Fatalf("mounts count = %d, want 2", len(mounts))
	}
		// 命名卷 source 应映射为真实卷名
		if string(mounts[0].Type) != "volume" || mounts[0].Source != "proj_mydata" {
		t.Errorf("volume mount = %+v, want type volume source proj_mydata", mounts[0])
	}
	if string(mounts[1].Type) != "tmpfs" || mounts[1].Target != "/tmp" {
		t.Errorf("tmpfs mount = %+v, want type tmpfs target /tmp", mounts[1])
	}
}

func TestTranslateVolumesBindMissingTarget(t *testing.T) {
	_, _, err := translateVolumes([]composeTypes.ServiceVolumeConfig{{Type: "bind", Source: "/x"}}, nil)
	if err == nil {
		t.Error("expected error for bind mount missing target")
	}
}

func TestApplyResourcesDeployLimits(t *testing.T) {
	var res container.Resources
	svc := composeTypes.ServiceConfig{
		Deploy: &composeTypes.DeployConfig{
			Resources: composeTypes.Resources{
				Limits: &composeTypes.Resource{
					NanoCPUs:    composeTypes.NanoCPUs(1.5),
					MemoryBytes: composeTypes.UnitBytes(512 * 1024 * 1024),
				},
			},
		},
	}
	applyResources(&res, svc)
	if res.Memory != 512*1024*1024 {
		t.Errorf("Memory = %d, want %d", res.Memory, 512*1024*1024)
	}
	if res.NanoCPUs != int64(1.5*1e9) {
		t.Errorf("NanoCPUs = %d, want %d", res.NanoCPUs, int64(1.5*1e9))
	}
}

func TestApplyResourcesLegacyFields(t *testing.T) {
	var res container.Resources
	svc := composeTypes.ServiceConfig{
		MemLimit: composeTypes.UnitBytes(256 * 1024 * 1024),
		CPUS:     2,
		CPUSet:   "0,1",
	}
	applyResources(&res, svc)
	if res.Memory != 256*1024*1024 {
		t.Errorf("Memory = %d, want %d", res.Memory, 256*1024*1024)
	}
	if res.NanoCPUs != int64(2*1e9) {
		t.Errorf("NanoCPUs = %d, want %d", res.NanoCPUs, int64(2*1e9))
	}
	if res.CpusetCpus != "0,1" {
		t.Errorf("CpusetCpus = %q, want 0,1", res.CpusetCpus)
	}
}

func TestTranslateHealthcheck(t *testing.T) {
	if translateHealthcheck(nil) != nil {
		t.Error("nil healthcheck should return nil")
	}
	disabled := translateHealthcheck(&composeTypes.HealthCheckConfig{Disable: true})
	if disabled == nil || len(disabled.Test) != 1 || disabled.Test[0] != "NONE" {
		t.Errorf("disabled healthcheck = %+v, want Test [NONE]", disabled)
	}
	interval := composeTypes.Duration(30 * time.Second)
	retries := uint64(3)
	hc := translateHealthcheck(&composeTypes.HealthCheckConfig{
		Test:     composeTypes.HealthCheckTest{"CMD", "curl", "-f", "http://localhost"},
		Interval: &interval,
		Retries:  &retries,
	})
	if hc == nil {
		t.Fatal("expected non-nil healthcheck")
	}
	if hc.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", hc.Interval)
	}
	if hc.Retries != 3 {
		t.Errorf("Retries = %d, want 3", hc.Retries)
	}
	if len(hc.Test) != 4 {
		t.Errorf("Test len = %d, want 4", len(hc.Test))
	}
}

func TestBuildLabels(t *testing.T) {
	svc := composeTypes.ServiceConfig{
		Labels: composeTypes.Labels{"custom": "value"},
	}
	labels := buildLabels("myproj", "/root/proj", "web", svc)
	if labels[labelProject] != "myproj" {
		t.Errorf("project label = %q, want myproj", labels[labelProject])
	}
	if labels[labelService] != "web" {
		t.Errorf("service label = %q, want web", labels[labelService])
	}
	if labels[labelWorkingDir] != "/root/proj" {
		t.Errorf("working_dir label = %q, want /root/proj", labels[labelWorkingDir])
	}
	if labels["custom"] != "value" {
		t.Errorf("custom label = %q, want value", labels["custom"])
	}
	if labels[labelConfigHash] == "" {
		t.Error("config-hash label should be set")
	}
}

func TestServiceConfigHashStable(t *testing.T) {
	svc := composeTypes.ServiceConfig{Image: "nginx", ContainerName: "web"}
	h1 := serviceConfigHash(svc)
	h2 := serviceConfigHash(svc)
	if h1 == "" || h1 != h2 {
		t.Errorf("hash not stable: %q vs %q", h1, h2)
	}
	svc.Image = "nginx:alpine"
	if serviceConfigHash(svc) == h1 {
		t.Error("hash should change when config changes")
	}
}

func TestTopoSort(t *testing.T) {
	services := composeTypes.Services{
		"web": composeTypes.ServiceConfig{
			DependsOn: composeTypes.DependsOnConfig{"db": composeTypes.ServiceDependency{}},
		},
		"db": composeTypes.ServiceConfig{},
	}
	order, err := topoSort(services)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("order len = %d, want 2", len(order))
	}
		// db 须排在 web 之前
		dbIdx, webIdx := -1, -1
	for i, name := range order {
		if name == "db" {
			dbIdx = i
		}
		if name == "web" {
			webIdx = i
		}
	}
	if dbIdx > webIdx {
		t.Errorf("db (%d) should come before web (%d)", dbIdx, webIdx)
	}
}

func TestTopoSortCycle(t *testing.T) {
	services := composeTypes.Services{
		"a": composeTypes.ServiceConfig{DependsOn: composeTypes.DependsOnConfig{"b": composeTypes.ServiceDependency{}}},
		"b": composeTypes.ServiceConfig{DependsOn: composeTypes.DependsOnConfig{"a": composeTypes.ServiceDependency{}}},
	}
	if _, err := topoSort(services); err == nil {
		t.Error("expected error for cyclic dependency")
	}
}

func TestTranslateServiceHostNetworkSkipsPorts(t *testing.T) {
	svc := composeTypes.ServiceConfig{
		Image:       "jrohy/webssh:latest",
		NetworkMode: "host",
		Ports:       []composeTypes.ServicePortConfig{{Target: 5032, Published: "5032"}},
	}
	tr, err := translateService("proj", "/root", "webssh", svc, "", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tr.hostConfig.PortBindings) != 0 {
		t.Errorf("host network should skip port bindings, got %v", tr.hostConfig.PortBindings)
	}
	if string(tr.hostConfig.NetworkMode) != "host" {
		t.Errorf("expected host network mode, got %q", tr.hostConfig.NetworkMode)
	}
		// host 模式不应挂默认网络
		if tr.network != nil {
		t.Errorf("host mode should have nil network config, got %+v", tr.network)
	}
}
