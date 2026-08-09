package utiles

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
)

func TestAggregatePullPercentage(t *testing.T) {
	layers := map[string]pullLayerProgress{
		"one": {Current: 25, Total: 100},
		"two": {Done: true},
	}
	if got := aggregatePullPercentage(layers); got != 63 {
		t.Fatalf("aggregatePullPercentage() = %d, want 63", got)
	}

	layers["one"] = pullLayerProgress{Current: 200, Total: 100}
	if got := aggregatePullPercentage(layers); got != 99 {
		t.Fatalf("aggregatePullPercentage() = %d, want 99", got)
	}
}

func TestAggregatePullBytes(t *testing.T) {
	layers := map[string]pullLayerProgress{
		"one":    {Current: 512, Total: 1024},
		"two":    {Current: 2048, Total: 4096},
		"cached": {Done: true},
	}
	current, total := aggregatePullBytes(layers)
	if current != 2560 || total != 5120 {
		t.Fatalf("aggregatePullBytes() = %d/%d, want 2560/5120", current, total)
	}
	if got := formatPullProgress(PullProgress{Status: "Downloading", Candidate: "nginx", Percentage: 50, Current: 2560, Total: 5120}); got != "Downloading · nginx · 拉取进度 50% · 2.5 KB/5.0 KB" {
		t.Fatalf("formatPullProgress() = %q", got)
	}
}
func TestDrainPullStreamWithProgress(t *testing.T) {
	stream := strings.NewReader(`
{"status":"Downloading","id":"layer-a","progressDetail":{"current":25,"total":100}}
{"status":"Already exists","id":"layer-b"}
{"status":"Pull complete","id":"layer-a"}
`)
	var updates []PullProgress
	if err := drainPullStreamWithProgress(stream, "nginx:latest", func(progress PullProgress) {
		updates = append(updates, progress)
	}); err != nil {
		t.Fatalf("drainPullStreamWithProgress() error = %v", err)
	}
	if len(updates) != 3 {
		t.Fatalf("got %d progress updates, want 3", len(updates))
	}
	if updates[0].Percentage != 25 {
		t.Fatalf("first percentage = %d, want 25", updates[0].Percentage)
	}
	if updates[0].Current != 25 || updates[0].Total != 100 {
		t.Fatalf("first size = %d/%d, want 25/100", updates[0].Current, updates[0].Total)
	}
	if updates[1].Percentage != 63 || updates[1].Status != "Already exists" {
		t.Fatalf("cached layer update = %+v, want 63%% and Already exists", updates[1])
	}
	if updates[2].Percentage != 100 {
		t.Fatalf("final percentage = %d, want 100", updates[2].Percentage)
	}
}

func TestDrainPullStreamWithProgressErrors(t *testing.T) {
	tests := []struct {
		name   string
		stream string
	}{
		{name: "invalid json", stream: `{"status":`},
		{name: "docker error", stream: `{"error":"pull denied","errorDetail":{"message":"pull denied"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := drainPullStreamWithProgress(strings.NewReader(tt.stream), "", nil); err == nil {
				t.Fatal("expected pull stream error")
			}
		})
	}
}

func TestPullImageRequiresSuccessfulLocalTag(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "settings.json"))
	if _, err := settingstore.SetHubURLs([]string{"mirror.example"}); err != nil {
		t.Fatal(err)
	}
	var paths []string
	pullCount := 0
	transport := pullRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		switch {
		case strings.HasSuffix(req.URL.Path, "/images/create"):
			pullCount++
			if pullCount == 1 {
				return pullResponse(http.StatusOK, `{"status":"Pull complete","id":"layer"}`), nil
			}
			return pullResponse(http.StatusInternalServerError, `{"message":"pull failed"}`), nil
		case strings.HasSuffix(req.URL.Path, "/tag"):
			return pullResponse(http.StatusInternalServerError, `{"message":"tag failed"}`), nil
		default:
			return pullResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		}
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.25"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	_, err = PullImage(context.Background(), cli, "nginx:latest", nil)
	if err == nil || !strings.Contains(err.Error(), "本地标签失败") {
		t.Fatalf("PullImage error = %v, want local tag failure", err)
	}
	if len(paths) < 2 || !strings.HasSuffix(paths[0], "/images/create") || !strings.HasSuffix(paths[1], "/tag") {
		t.Fatalf("unexpected request sequence: %v", paths)
	}
}

func TestPullImageStopsAfterContextCancellation(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "settings.json"))
	if _, err := settingstore.SetHubURLs([]string{"mirror.example", "mirror-two.example"}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	transport := pullRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		cancel()
		return nil, context.Canceled
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.25"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	_, err = PullImage(ctx, cli, "nginx:latest", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PullImage error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("pull attempts = %d, want 1", calls)
	}
}

func pullResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewBufferString(body))}
}

type pullRoundTripFunc func(*http.Request) (*http.Response, error)

func (f pullRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestPullTaskPercentage(t *testing.T) {
	if got := pullTaskPercentage(50, 10, 30); got != 20 {
		t.Fatalf("pullTaskPercentage() = %d, want 20", got)
	}
	if got := pullTaskPercentage(-10, 10, 30); got != 10 {
		t.Fatalf("negative pull percentage = %d, want 10", got)
	}
	if got := pullTaskPercentage(120, 10, 30); got != 30 {
		t.Fatalf("over-100 pull percentage = %d, want 30", got)
	}
}
