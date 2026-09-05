package utiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertOfficialUpdateURL(t *testing.T) {
	t.Setenv("updateRepo", "")
	ok := []string{
		"https://raw.githubusercontent.com/syueya/dockerCopilot/latest/version",
		"https://github.com/syueya/dockerCopilot/releases/download/v2.2.0/dockerCopilot-amd64.tar.gz",
		"https://github.com/syueya/dockerCopilot/releases/download/v2.2.0/dockerCopilot-amd64.tar.gz.sha256",
	}
	for _, u := range ok {
		if err := assertOfficialUpdateURL(u); err != nil {
			t.Fatalf("expected ok %s: %v", u, err)
		}
	}
	bad := []string{
		"http://github.com/syueya/dockerCopilot/releases/download/v1/dockerCopilot-amd64.tar.gz",
		"https://evil.com/syueya/dockerCopilot/latest/version",
		"https://raw.githubusercontent.com/onlyLTY/dockerCopilot/latest/version",
		"https://github.com/syueya/dockerCopilot/releases/download/../v1/x",
		"https://github.com/syueya/dockerCopilot/releases/download/v1/dockerCopilot-amd64.tar.gz?x=1",
	}
	for _, u := range bad {
		if err := assertOfficialUpdateURL(u); err == nil {
			t.Fatalf("expected reject %s", u)
		}
	}
}

func TestUpdateRepoOverride(t *testing.T) {
	t.Setenv("updateRepo", "alice/my-copilot.git_1")
	u, err := OfficialVersionURL()
	if err != nil {
		t.Fatal(err)
	}
	want := "https://raw.githubusercontent.com/alice/my-copilot.git_1/latest/version"
	if u != want {
		t.Fatalf("got %s want %s", u, want)
	}
	if err := assertOfficialUpdateURL(want); err != nil {
		t.Fatalf("expected ok %s: %v", want, err)
	}
	t.Setenv("updateRepo", "not a repo")
	if _, err := OfficialVersionURL(); err == nil {
		t.Fatal("expected invalid updateRepo reject")
	}
}

func TestWithGithubProxy(t *testing.T) {
	t.Setenv("updateRepo", "")
	official := "https://github.com/syueya/dockerCopilot/releases/download/v2.2.0/dockerCopilot-amd64.tar.gz"
	got, err := withGithubProxy("", official)
	if err != nil || got != official {
		t.Fatalf("empty proxy: got %q err=%v", got, err)
	}
	got, err = withGithubProxy("https://mirror.example/gh", official)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "https://mirror.example/gh/") || !strings.HasSuffix(got, official) {
		t.Fatalf("unexpected proxied url: %s", got)
	}
	if _, err := withGithubProxy("ftp://mirror.example", official); err == nil {
		t.Fatal("expected reject ftp proxy")
	}
	if _, err := withGithubProxy("https://user:pass@mirror.example", official); err == nil {
		t.Fatal("expected reject userinfo proxy")
	}
}

func TestOfficialReleaseAssetURL(t *testing.T) {
	t.Setenv("githubProxy", "")
	t.Setenv("updateRepo", "")
	u, err := OfficialReleaseAssetURL("v2.2.0", "dockerCopilot-amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/syueya/dockerCopilot/releases/download/v2.2.0/dockerCopilot-amd64.tar.gz"
	if u != want {
		t.Fatalf("got %s want %s", u, want)
	}
	if _, err := OfficialReleaseAssetURL("../v2", "x.tar.gz"); err == nil {
		t.Fatal("expected bad version reject")
	}
	if _, err := OfficialReleaseAssetURL("v2.2.0", "../x.tar.gz"); err == nil {
		t.Fatal("expected bad asset reject")
	}
}

func TestFindUpdateVersionFile(t *testing.T) {
	dir := t.TempDir()
	if got := findUpdateVersionFile(dir); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if err := os.MkdirAll(filepath.Join(dir, "web"), 0755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "version")
	if err := os.WriteFile(want, []byte("v2.2.6"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := findUpdateVersionFile(dir); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseSHA256File(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	cases := []struct {
		in   string
		ok   bool
		want string
	}{
		{sum, true, sum},
		{sum + "  dockerCopilot-amd64.tar.gz\n", true, sum},
		{sum + " *dockerCopilot-amd64.tar.gz", true, sum},
		{strings.ToUpper(sum) + "  file", true, sum},
		{"", false, ""},
		{"deadbeef", false, ""},
		{"not-hex-not-hex-not-hex-not-hex-not-hex-not-hex-not-hex-not-hex-zzzz", false, ""},
	}
	for _, tc := range cases {
		got, err := parseSHA256File(tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Fatalf("in=%q got=%q err=%v", tc.in, got, err)
			}
		} else if err == nil {
			t.Fatalf("expected error for %q", tc.in)
		}
	}
}

func TestFileSHA256(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/blob"
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	// echo -n hello | sha256sum
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if sum != want {
		t.Fatalf("got %s want %s", sum, want)
	}
}
