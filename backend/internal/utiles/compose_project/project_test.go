package compose_project

import (
	"os"
	"path/filepath"
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestContainerPortsMergesDockerPortSources(t *testing.T) {
	bound, _ := nat.NewPort("tcp", "8080")
	exposed, _ := nat.NewPort("tcp", "9090")
	inspect := dockerTypes.ContainerJSON{
		ContainerJSONBase: &dockerTypes.ContainerJSONBase{HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}}},
		Config:            &container.Config{ExposedPorts: nat.PortSet{exposed: struct{}{}}},
		NetworkSettings:   &dockerTypes.NetworkSettings{NetworkSettingsBase: dockerTypes.NetworkSettingsBase{Ports: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}}},
	}
	ports := containerPorts(inspect)
	if len(ports) != 2 {
		t.Fatalf("expected two unique ports, got %#v", ports)
	}
	if ports[0].ContainerPort != "8080" || !ports[0].Published {
		t.Fatalf("expected published 8080 mapping, got %#v", ports[0])
	}
}

func TestContainerPortsDeduplicatesExposedPortWhenPublished(t *testing.T) {
	bound, _ := nat.NewPort("tcp", "8080")
	inspect := dockerTypes.ContainerJSON{
		ContainerJSONBase: &dockerTypes.ContainerJSONBase{
			HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}},
		},
		Config: &container.Config{ExposedPorts: nat.PortSet{bound: struct{}{}}},
	}
	ports := containerPorts(inspect)
	if len(ports) != 1 || !ports[0].Published || ports[0].HostPort != "18080" {
		t.Fatalf("expected exposed port to be replaced by published binding, got %#v", ports)
	}
}

func TestContainerPortsIncludesHostBindingWithoutNetworkSettings(t *testing.T) {
	bound, _ := nat.NewPort("udp", "5353")
	inspect := dockerTypes.ContainerJSON{ContainerJSONBase: &dockerTypes.ContainerJSONBase{HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostPort: "15353"}}}}}}
	ports := containerPorts(inspect)
	if len(ports) != 1 || ports[0].ContainerPort != "5353" || ports[0].Protocol != "udp" {
		t.Fatalf("expected host binding fallback, got %#v", ports)
	}
}

func TestSameComposeRootUsesPathMapping(t *testing.T) {
	mapping := composePathMapping{source: "/data/docker-data", destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": "/data/docker-data/dockercopilot"}
	if !sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected host and container compose roots to match")
	}
}

func TestSameComposeRootDoesNotMatchSiblingPath(t *testing.T) {
	mapping := composePathMapping{source: "/data/docker-data", destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": "/data/docker-database/dockercopilot"}
	if sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected unrelated host root not to match")
	}
}

func TestSameComposeRootNormalizesWindowsSeparators(t *testing.T) {
	mapping := composePathMapping{source: `E:\\compose`, destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": `E:\\compose\\dockercopilot`}
	if !sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected Windows host root to match mapped container root")
	}
}

func TestSameComposeRootRequiresWorkingDirectory(t *testing.T) {
	if sameComposeRoot(map[string]string{}, "/compose/dockercopilot", nil) {
		t.Fatal("expected missing working directory not to match")
	}
}

func TestComposePathMappingsResolveRelativeHostPath(t *testing.T) {
	svcCtx := &svc.ServiceContext{}
	svcCtx.Config.Compose.PathMappings = []config.ComposePathMapping{{
		HostPath:      ".",
		ContainerPath: "/compose",
	}}
	mappings := composePathMappings(svcCtx, nil)
	if len(mappings) != 1 {
		t.Fatalf("expected one mapping, got %#v", mappings)
	}
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	if mappings[0].source != normalizeComposePath(root) || mappings[0].destination != "/compose" {
		t.Fatalf("unexpected mapping: %#v", mappings[0])
	}
	labels := map[string]string{"com.docker.compose.project.working_dir": root + string(filepath.Separator) + "project"}
	if !sameComposeRoot(labels, "/compose/project", mappings) {
		t.Fatal("expected relative host mapping to match")
	}
}

func TestComposeMountIsRelevantWithRelativeScanPath(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	if !composeMountIsRelevant(root, "/compose", []string{"."}) {
		t.Fatal("expected relative scan path to match host mount source")
	}
}

func TestRemoveProjectRootRemovesNestedContent(t *testing.T) {
	scanRoot := t.TempDir()
	projectRoot := filepath.Join(scanRoot, "project")
	if err := os.MkdirAll(filepath.Join(projectRoot, "nested", "deep"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "compose.yaml"), []byte("services: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "nested", "deep", "data.txt"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveProjectRoot(projectRoot, []string{scanRoot}); err != nil {
		t.Fatalf("remove project root: %v", err)
	}
	if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
		t.Fatalf("expected project root to be removed, got %v", err)
	}
}

func TestRemoveProjectRootRejectsScanRoot(t *testing.T) {
	scanRoot := t.TempDir()
	if err := RemoveProjectRoot(scanRoot, []string{scanRoot}); err == nil {
		t.Fatal("expected scan root removal to be rejected")
	}
	if _, err := os.Stat(scanRoot); err != nil {
		t.Fatalf("scan root should remain: %v", err)
	}
}

func TestRemoveProjectRootRejectsOutsidePath(t *testing.T) {
	scanRoot := t.TempDir()
	outsideRoot := t.TempDir()
	if err := RemoveProjectRoot(outsideRoot, []string{scanRoot}); err == nil {
		t.Fatal("expected outside path removal to be rejected")
	}
	if _, err := os.Stat(outsideRoot); err != nil {
		t.Fatalf("outside path should remain: %v", err)
	}
}

func TestRemoveProjectRootRejectsSymlinkRoot(t *testing.T) {
	scanRoot := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(scanRoot, "project-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if err := RemoveProjectRoot(link, []string{scanRoot}); err == nil {
		t.Fatal("expected symlink project root removal to be rejected")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target should remain: %v", err)
	}
}
