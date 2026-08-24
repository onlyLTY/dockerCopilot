package module

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func TestCompareRepoDigests(t *testing.T) {
	tests := []struct {
		name       string
		local      []string
		remote     string
		needUpdate bool
		comparable bool
	}{
		{
			name:       "single digest matches",
			local:      []string{"postgres@sha256:current"},
			remote:     "sha256:current",
			needUpdate: false,
			comparable: true,
		},
		{
			name:       "single digest differs",
			local:      []string{"postgres@sha256:old"},
			remote:     "sha256:current",
			needUpdate: true,
			comparable: true,
		},
		{
			name: "matching digest is not last",
			local: []string{
				"postgres@sha256:current",
				"mirror.example/postgres@sha256:mirror",
			},
			remote:     "sha256:current",
			needUpdate: false,
			comparable: true,
		},
		{
			name: "matching digest is last",
			local: []string{
				"mirror.example/postgres@sha256:mirror",
				"postgres@sha256:current",
			},
			remote:     "sha256:current",
			needUpdate: false,
			comparable: true,
		},
		{
			name:       "malformed local digest is ignored",
			local:      []string{"postgres:17-alpine", "postgres@sha256:current"},
			remote:     "sha256:current",
			needUpdate: false,
			comparable: true,
		},
		{
			name:       "no usable local digest",
			local:      []string{"postgres:17-alpine"},
			remote:     "sha256:current",
			needUpdate: false,
			comparable: false,
		},
		{
			name:       "empty remote digest",
			local:      []string{"postgres@sha256:current"},
			remote:     "",
			needUpdate: false,
			comparable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needUpdate, comparable := compareRepoDigests(tt.local, tt.remote)
			if needUpdate != tt.needUpdate || comparable != tt.comparable {
				t.Fatalf(
					"compareRepoDigests() = (%v, %v), want (%v, %v)",
					needUpdate,
					comparable,
					tt.needUpdate,
					tt.comparable,
				)
			}
		})
	}
}

func TestImageUpdateDataMarkCurrent(t *testing.T) {
	data := NewImageCheck()
	data.data["sha256:image"] = ImageCheckList{NeedUpdate: true}

	data.MarkCurrent("sha256:image")

	if data.NeedUpdate("sha256:image") {
		t.Fatal("expected MarkCurrent to clear the cached update state")
	}
}

func TestExpandImageReferencesSkipsDigestOnlyReferences(t *testing.T) {
	const digestReference = "ghcr.io/autunn/dockercopilot@sha256:3667bdb9f23780de257a2105755c82e1633afc01e98189e7270c3c5a9b3ee30e"
	if !isDigestOnlyReference(digestReference) {
		t.Fatal("expected digest-only reference to be recognized")
	}

	expanded := expandImageReferences([]types.Image{
		{Reference: digestReference},
		{Reference: "postgres:17-alpine"},
	})
	if len(expanded) != 1 {
		t.Fatalf("expected only the tagged reference, got %+v", expanded)
	}
	if expanded[0].Reference != "docker.io/library/postgres:17-alpine" {
		t.Fatalf("unexpected normalized reference %q", expanded[0].Reference)
	}
}

func TestCheckSingleImageUsesDockerDaemonDigest(t *testing.T) {
	const currentDigest = "sha256:18cfe3ef5e6815560c98237d6216d1e5119702fb0f3894c8785dd58b8bbe5d73"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.47/distribution/docker.io/library/postgres:17-alpine/json" {
			t.Errorf("unexpected Docker API path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Descriptor": {
				"mediaType": "application/vnd.oci.image.index.v1+json",
				"digest": "` + currentDigest + `",
				"size": 1024
			},
			"Platforms": [
				{"architecture": "amd64", "os": "linux"},
				{"architecture": "arm64", "os": "linux", "variant": "v8"}
			]
		}`))
	}))
	defer server.Close()

	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(server.URL),
		client.WithVersion("1.47"),
		client.WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}

	needUpdate, comparable := checkSingleImage(context.Background(), dockerClient, types.Image{
		ImageName: "postgres",
		ImageTag:  "17-alpine",
		Summary: dockerimage.Summary{
			RepoDigests: []string{"postgres@" + currentDigest},
		},
	})
	if needUpdate || !comparable {
		t.Fatalf("checkSingleImage() = (%v, %v), want (false, true)", needUpdate, comparable)
	}
}
