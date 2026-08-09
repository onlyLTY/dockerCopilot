package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func TestRemoveImageUsesRepoTagForNonForceDelete(t *testing.T) {
	testRemoveImageTarget(t, types.RemoveImageReq{
		IdReq:   types.IdReq{Id: "sha256:image"},
		RepoTag: "docker.m.daocloud.io/openlistteam/openlist:latest",
	}, "/images/docker.m.daocloud.io/openlistteam/openlist:latest", "")
}

func TestRemoveImageFallsBackToID(t *testing.T) {
	testRemoveImageTarget(t, types.RemoveImageReq{
		IdReq: types.IdReq{Id: "sha256:image"},
	}, "/images/sha256:image", "")
}

func TestRemoveImageUsesIDForForceDelete(t *testing.T) {
	testRemoveImageTarget(t, types.RemoveImageReq{
		IdReq: types.IdReq{Id: "sha256:image"},
		Force: true,
	}, "/images/sha256:image", "1")
}

func testRemoveImageTarget(t *testing.T, req types.RemoveImageReq, wantPath, wantForce string) {
	t.Helper()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete {
			return nil, fmt.Errorf("method = %s, want DELETE", req.Method)
		}
		if req.URL.Path != "/v1.25"+wantPath {
			return nil, fmt.Errorf("path = %s, want %s", req.URL.Path, "/v1.25"+wantPath)
		}
		if got := req.URL.Query().Get("force"); got != wantForce {
			return nil, fmt.Errorf("force = %q, want %q", got, wantForce)
		}
		body, err := json.Marshal([]image.DeleteResponse{{Untagged: "example/app:latest"}})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
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

	resp, err := NewRemoveLogic(context.Background(), &svc.ServiceContext{DockerClient: dockerClient}).Remove(&req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200", resp.Code)
	}
}
