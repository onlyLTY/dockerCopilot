package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestImagesListReturnsAllRepoTags(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body any
		switch req.URL.Path {
		case "/v1.25/images/json":
			body = []image.Summary{{
				ID:       "sha256:image",
				RepoTags: []string{"example/app:latest", "example/app:v1"},
				Created:  1,
			}}
		case "/v1.25/containers/json":
			body = []struct{}{}
		default:
			t.Fatalf("unexpected request path %s", req.URL.Path)
		}
		content, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(content)),
		}, nil
	})
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost("http://docker.test"),
		client.WithVersion("1.25"),
		client.WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer dockerClient.Close()

	resp, err := NewImagesListLogic(context.Background(), &svc.ServiceContext{DockerClient: dockerClient}).ImagesList()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200", resp.Code)
	}
	items, ok := resp.Data.([]imageListItem)
	if !ok {
		t.Fatalf("response data type = %T, want []imageListItem", resp.Data)
	}
	if len(items) != 1 {
		t.Fatalf("image count = %d, want 1", len(items))
	}
	if got, want := items[0].RepoTags, []string{"example/app:latest", "example/app:v1"}; !equalStrings(got, want) {
		t.Fatalf("repoTags = %v, want %v", got, want)
	}
	if items[0].Name != "example/app" || items[0].Tag != "latest" {
		t.Fatalf("display image = %s:%s, want example/app:latest", items[0].Name, items[0].Tag)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
func TestImageNameForDisplay(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		hubURLs []string
		want    string
	}{
		{
			name:    "removes configured accelerator host",
			image:   "docker.1ms.run/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "library/nginx",
		},
		{
			name:    "matches host case insensitively",
			image:   "DOCKER.1MS.RUN/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "library/nginx",
		},
		{
			name:    "removes the first matching configured host",
			image:   "docker.m.daocloud.io/library/redis",
			hubURLs: []string{"docker.1ms.run", "docker.m.daocloud.io"},
			want:    "library/redis",
		},
		{
			name:    "keeps similar host",
			image:   "docker.1ms.run-extra/library/nginx",
			hubURLs: []string{"docker.1ms.run"},
			want:    "docker.1ms.run-extra/library/nginx",
		},
		{
			name:    "keeps unconfigured registry",
			image:   "ghcr.io/example/image",
			hubURLs: []string{"docker.1ms.run"},
			want:    "ghcr.io/example/image",
		},
		{
			name:    "ignores empty and slash wrapped hosts",
			image:   "docker.1ms.run/library/nginx",
			hubURLs: []string{"", "/docker.1ms.run/"},
			want:    "library/nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageNameForDisplay(tt.image, tt.hubURLs); got != tt.want {
				t.Fatalf("imageNameForDisplay(%q, %v) = %q, want %q", tt.image, tt.hubURLs, got, tt.want)
			}
		})
	}
}
