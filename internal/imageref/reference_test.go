package imageref

import "testing"

func TestParseTaggedSupportsRegistryPort(t *testing.T) {
	parsed, err := ParseTagged("registry.example:5000/team/postgres:17-alpine")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Normalized != "registry.example:5000/team/postgres:17-alpine" || parsed.Repository != "registry.example:5000/team/postgres" || parsed.Tag != "17-alpine" {
		t.Fatalf("unexpected parsed reference: %+v", parsed)
	}
}

func TestCacheKeyNormalizesDockerHubReference(t *testing.T) {
	if got := CacheKey("postgres:17-alpine"); got != "docker.io/library/postgres:17-alpine" {
		t.Fatalf("unexpected cache key %q", got)
	}
}

func TestRepositoryKeyDropsTagAndDigest(t *testing.T) {
	for _, value := range []string{
		"postgres:17-alpine",
		"docker.io/library/postgres@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	} {
		got, err := RepositoryKey(value)
		if err != nil {
			t.Fatal(err)
		}
		if got != "docker.io/library/postgres" {
			t.Fatalf("unexpected repository key %q for %q", got, value)
		}
	}
}
