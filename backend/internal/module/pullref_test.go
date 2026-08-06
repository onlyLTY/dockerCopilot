package module

import (
	"path/filepath"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
)

func TestResolvePullCandidatesOfficialWithHubURLs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(dir, "appSettings.json"))
	if _, err := settingstore.SetHubURLs([]string{"docker.m.daocloud.io", "docker.1ms.run"}); err != nil {
		t.Fatal(err)
	}

	cands, localName, err := ResolvePullCandidates("nginx")
	if err != nil {
		t.Fatal(err)
	}
	if localName != "nginx" {
		t.Fatalf("localName = %q", localName)
	}
	if len(cands) < 3 {
		t.Fatalf("candidates = %v", cands)
	}
	if cands[0] != "docker.m.daocloud.io/library/nginx:latest" {
		t.Fatalf("first candidate = %q", cands[0])
	}
	if cands[1] != "docker.1ms.run/library/nginx:latest" {
		t.Fatalf("second candidate = %q", cands[1])
	}
	// 原始引用在加速源之后
	foundOriginal := false
	for _, c := range cands[2:] {
		if c == "nginx" {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Fatalf("original ref missing in %v", cands)
	}
}

func TestResolvePullCandidatesPrivateUnchanged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(dir, "appSettings.json"))
	if _, err := settingstore.SetHubURLs([]string{"docker.m.daocloud.io"}); err != nil {
		t.Fatal(err)
	}

	cands, localName, err := ResolvePullCandidates("ghcr.io/foo/bar:1.2")
	if err != nil {
		t.Fatal(err)
	}
	if localName != "ghcr.io/foo/bar:1.2" || len(cands) != 1 || cands[0] != localName {
		t.Fatalf("private image rewritten: localName=%q cands=%v", localName, cands)
	}
}

func TestResolvePullCandidatesEmptyHubURLs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(dir, "appSettings.json"))
	if _, err := settingstore.SetHubURLs(nil); err != nil {
		t.Fatal(err)
	}

	cands, localName, err := ResolvePullCandidates("library/redis:7")
	if err != nil {
		t.Fatal(err)
	}
	if localName != "library/redis:7" {
		t.Fatalf("localName = %q", localName)
	}
	// 无加速源时不应出现 mirror 前缀
	for _, c := range cands {
		if c != "library/redis:7" && c != "redis:7" && c != "docker.io/library/redis:7" {
			t.Fatalf("unexpected candidate %q in %v", c, cands)
		}
	}
}

func TestResolvePullCandidatesUserImage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(dir, "appSettings.json"))
	if _, err := settingstore.SetHubURLs([]string{"hub.rat.dev"}); err != nil {
		t.Fatal(err)
	}

	// Docker 引用要求小写；大写会被 reference 库误解析为 domain
	cands, localName, err := ResolvePullCandidates("onlylty/demo:1.0")
	if err != nil {
		t.Fatal(err)
	}
	if localName != "onlylty/demo:1.0" {
		t.Fatalf("localName = %q", localName)
	}
	if cands[0] != "hub.rat.dev/onlylty/demo:1.0" {
		t.Fatalf("first candidate = %q, full=%v", cands[0], cands)
	}
	found := false
	for _, c := range cands {
		if c == "onlylty/demo:1.0" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("original missing: %v", cands)
	}
}
