package icons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRepository(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "tag", input: "nginx:1.25", want: "nginx"},
		{name: "digest", input: "nginx@sha256:abcdef", want: "nginx"},
		{name: "docker hub", input: "docker.io/library/nginx:latest", want: "library/nginx"},
		{name: "library name", input: "library/nginx", want: "library/nginx"},
		{name: "registry path", input: "ghcr.io/company/app:2.0", want: "company/app"},
		{name: "registry port", input: "localhost:5000/team/app:latest", want: "team/app"},
		{name: "vaultwarden server", input: "vaultwarden/server:latest", want: "vaultwarden/server"},
		{name: "punctuation", input: "registry.local/team/foo-bar_baz.v2:latest", want: "team/foo-bar_baz.v2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeRepository(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("NormalizeRepository(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeRepositoryRejectsUnsafeValues(t *testing.T) {
	for _, input := range []string{"", "../app", "registry/app name", "registry/app\nname", "registry/\"app"} {
		if _, err := NormalizeRepository(input); err == nil {
			t.Fatalf("NormalizeRepository(%q) accepted unsafe value", input)
		}
	}
}

func TestReadIconsSupportsLegacyJavaScriptConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "imageLogos.js")
	content := "// legacy config\nexport const customImageLogos = {\n  \"nginx:latest\": \"/src/config/image/nginx.png\",\n  \"ghcr.io/company/app\": \"/src/config/image/app.png\",\n};\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	icons, err := readIcons(path)
	if err != nil {
		t.Fatal(err)
	}
	if icons["nginx:latest"] != "/src/config/image/nginx.png" || icons["ghcr.io/company/app"] != "/src/config/image/app.png" {
		t.Fatalf("unexpected icons: %#v", icons)
	}
}

func TestWriteIconsEscapesValuesAndCanBeReadBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "imageLogos.js")
	want := map[string]string{
		"ghcr.io/company/foo-bar": "/src/config/image/foo.png",
		"registry.local/team/app": "/src/config/image/app-\\\"x.png",
	}
	if err := writeIcons(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readIcons(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("read back %d icons, want %d: %#v", len(got), len(want), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("read back %q = %q, want %q", key, got[key], value)
		}
	}
}

func TestMatchingKeysUsesNormalizedRepositoryNames(t *testing.T) {
	iconsMap := map[string]string{
		"nginx:latest":       "/src/config/image/docker.png",
		"docker.io/nginx:1":  "/src/config/image/docker-old.png",
		"ghcr.io/team/nginx": "/src/config/image/ghcr.png",
	}
	keys := matchingKeys(iconsMap, "nginx")
	if len(keys) != 2 {
		t.Fatalf("matchingKeys returned %v, want two normalized nginx keys", keys)
	}
	if got := matchingKeys(iconsMap, "team/nginx"); len(got) != 1 || got[0] != "ghcr.io/team/nginx" {
		t.Fatalf("matchingKeys returned %v for normalized private registry path", got)
	}
}
