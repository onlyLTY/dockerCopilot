package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func TestCountRemovedImages(t *testing.T) {
	before := []image.Summary{
		{ID: "sha256:kept"},
		{ID: "sha256:removed"},
		{ID: "sha256:removed-again"},
	}
	after := []image.Summary{{ID: "sha256:kept"}}

	if got := countRemovedImages(before, after); got != 2 {
		t.Fatalf("countRemovedImages() = %d, want 2", got)
	}
}

func TestPruneImagesCountsImageDiff(t *testing.T) {
	requests := make([]*http.Request, 0, 3)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Clone(req.Context()))
		var body any
		switch req.URL.Path {
		case "/v1.25/images/json":
			if len(requests) == 1 {
				body = []image.Summary{{ID: "sha256:kept"}, {ID: "sha256:removed"}}
			} else {
				body = []image.Summary{{ID: "sha256:kept"}}
			}
		case "/v1.25/images/prune":
			body = image.PruneReport{
				ImagesDeleted: []image.DeleteResponse{
					{Untagged: "example:old"},
					{Deleted: "sha256:removed"},
					{Deleted: "sha256:layer"},
				},
				SpaceReclaimed: 123,
			}
		default:
			return nil, nil
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

	resp, err := NewPruneImagesLogic(context.Background(), &svc.ServiceContext{DockerClient: dockerClient}).PruneImages(&types.ImagePruneReq{Kind: "untagged"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d, want 200", resp.Code)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("response data type = %T, want map[string]interface{}", resp.Data)
	}
	if got := data["deleted"]; got != 1 {
		t.Fatalf("deleted = %v, want 1", got)
	}
	if got := data["spaceReclaimed"]; got != uint64(123) {
		t.Fatalf("spaceReclaimed = %v, want 123", got)
	}
	if len(requests) != 3 {
		t.Fatalf("request count = %d, want 3", len(requests))
	}
	if requests[0].Method != http.MethodGet || requests[1].Method != http.MethodPost || requests[2].Method != http.MethodGet {
		t.Fatalf("request sequence = %s, %s, %s, want GET, POST, GET", requests[0].Method, requests[1].Method, requests[2].Method)
	}
	if !strings.Contains(requests[1].URL.Query().Get("filters"), `"dangling":{"true"`) {
		t.Fatalf("prune filters = %q, want dangling=true", requests[1].URL.Query().Get("filters"))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
