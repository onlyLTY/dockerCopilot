package utiles

import "testing"

func TestIsSensitiveHostPath(t *testing.T) {
	sensitive := []string{"/", "/etc", "/etc/passwd", "/proc", "/proc/1", "/sys", "/sys/fs", "/var/run", "/var/run/docker.sock"}
	for _, p := range sensitive {
		if !IsSensitiveHostPath(p) {
			t.Fatalf("expected sensitive: %s", p)
		}
	}
	safe := []string{"/data", "/compose/app", "/home/user", "/var/lib/docker/volumes/x"}
	for _, p := range safe {
		if IsSensitiveHostPath(p) {
			t.Fatalf("expected not sensitive: %s", p)
		}
	}
}

func TestIsDockerSocketPath(t *testing.T) {
	if !IsDockerSocketPath("/var/run/docker.sock") {
		t.Fatal("expected docker.sock match")
	}
	if !IsDockerSocketPath("/run/docker.sock") {
		t.Fatal("expected docker.sock match")
	}
	if IsDockerSocketPath("/data/app.sock") {
		t.Fatal("should not match unrelated sock")
	}
}

func TestBindSource(t *testing.T) {
	if got := BindSource("/host/path:/container/path:ro"); got != "/host/path" {
		t.Fatalf("got %q", got)
	}
	if got := BindSource("/only"); got != "/only" {
		t.Fatalf("got %q", got)
	}
}
