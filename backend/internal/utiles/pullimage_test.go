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
		"two": {Current: 50, Total: 100, Done: true},
	}
	if got := aggregatePullPercentage(layers); got != 38 {
		t.Fatalf("aggregatePullPercentage() = %d, want 38", got)
	}

	layers["one"] = pullLayerProgress{Current: 200, Total: 100}
	if got := aggregatePullPercentage(layers); got != 99 {
		t.Fatalf("aggregatePullPercentage() = %d, want 99", got)
	}
}

func TestDrainPullStreamPercentageMatchesBytes(t *testing.T) {
	stream := strings.NewReader(`
	{"status":"Downloading","id":"layer-a","progressDetail":{"current":80,"total":100}}
	{"status":"Downloading","id":"layer-b","progressDetail":{"current":0,"total":900}}
	{"status":"Downloading","id":"layer-b","progressDetail":{"current":100,"total":900}}
	`)
	var updates []PullProgress
	if err := drainPullStreamWithProgress(stream, "nginx:latest", func(progress PullProgress) {
		if progress.Status == pullStageDownloading {
			updates = append(updates, progress)
		}
	}); err != nil {
		t.Fatalf("drainPullStreamWithProgress() error = %v", err)
	}
	for _, update := range updates {
		if update.Total <= 0 {
			continue
		}
		want := int(float64(update.Current)/float64(update.Total)*100 + 0.5)
		if update.Percentage != want {
			t.Fatalf("percentage %d does not match %d/%d = %d%%", update.Percentage, update.Current, update.Total, want)
		}
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
	if got := formatPullProgress(PullProgress{Status: "正在下载镜像", Candidate: "nginx", Percentage: 50, Current: 2560, Total: 5120}); got != "nginx" {
		t.Fatalf("formatPullProgress() = %q", got)
	}
}
func TestSummarizePullError(t *testing.T) {
	if got := summarizePullError(errors.New(`request failed: EOF`)); got != "连接中断（EOF）" {
		t.Fatalf("summarize EOF = %q", got)
	}
	if got := pullFailureReason("拉取流失败", errors.New("request failed: EOF")); got != "拉取流失败：连接中断（EOF）" {
		t.Fatalf("pull failure reason = %q", got)
	}
	if got := pullFailureDetail(2, 10, "拉取流失败", errors.New("request failed: EOF")); got != "第 2/10 个源失败：拉取流失败：连接中断（EOF）" {
		t.Fatalf("pull failure detail = %q", got)
	}
}

func TestDrainPullStreamReportsDownloadCompletion(t *testing.T) {
	stream := strings.NewReader(`
	{"status":"Downloading","id":"layer-a","progressDetail":{"current":80,"total":100}}
	{"status":"Download complete","id":"layer-a","progressDetail":{"current":80,"total":100}}
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
	if updates[1].Status != pullStageDownloading || updates[1].Percentage != 100 {
		t.Fatalf("download completion update = %+v", updates[1])
	}
	if updates[2].Status != pullStageDownloading || updates[2].Percentage != 100 {
		t.Fatalf("final download update = %+v", updates[2])
	}
}

func TestDrainPullStreamDoesNotExtractAfterSingleLayer(t *testing.T) {
	stream := strings.NewReader(`
	{"status":"Downloading","id":"layer-a","progressDetail":{"current":80,"total":100}}
	{"status":"Download complete","id":"layer-a","progressDetail":{"current":80,"total":100}}
	{"status":"Downloading","id":"layer-b","progressDetail":{"current":10,"total":900}}
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
	if updates[1].Status != pullStageDownloading || updates[1].Indeterminate {
		t.Fatalf("single layer completion update = %+v", updates[1])
	}
	if updates[2].Status != pullStageDownloading || updates[2].Indeterminate {
		t.Fatalf("late layer download update = %+v", updates[2])
	}
	if updates[2].Current != 110 || updates[2].Total != 1000 || updates[2].Percentage != 11 {
		t.Fatalf("late layer aggregate = %+v, want 110/1000 at 11%%", updates[2])
	}
}

func TestPullStageForLayersWaitsForAllDownloads(t *testing.T) {
	layers := map[string]pullLayerProgress{
		"done-download": {Status: "Download complete"},
		"waiting":       {Status: "Waiting"},
	}
	if got := pullStageForLayers(layers); got != pullStageWaiting {
		t.Fatalf("stage with waiting layer = %q, want %q", got, pullStageWaiting)
	}
	layers["waiting"] = pullLayerProgress{Status: "Downloading", Current: 10, Total: 100}
	if got := pullStageForLayers(layers); got != pullStageDownloading {
		t.Fatalf("stage with active download = %q, want %q", got, pullStageDownloading)
	}
	layers["waiting"] = pullLayerProgress{Status: "Download complete"}
	if got := pullStageForLayers(layers); got != pullStageExtracting {
		t.Fatalf("stage after all downloads = %q, want %q", got, pullStageExtracting)
	}
}
func TestPullStageReturnsToDownloadingForLateLayer(t *testing.T) {
	layers := map[string]pullLayerProgress{
		"extracting": {Status: "Extracting"},
	}
	if got := pullStageForLayers(layers); got != pullStageExtracting {
		t.Fatalf("initial extracting stage = %q, want %q", got, pullStageExtracting)
	}
	layers["late-layer"] = pullLayerProgress{Status: "Downloading", Current: 10, Total: 100}
	if got := pullStageForLayers(layers); got != pullStageDownloading {
		t.Fatalf("late download stage = %q, want %q", got, pullStageDownloading)
	}
}

func TestDrainPullStreamWithProgress(t *testing.T) {
	stream := strings.NewReader(`
	{"status":"Downloading","id":"layer-a","progressDetail":{"current":25,"total":100}}
	{"status":"Already exists","id":"layer-b"}
	{"status":"Download complete","id":"layer-a"}
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
	if updates[0].Percentage != 25 || updates[0].Status != "正在下载镜像" || updates[0].Current != 25 || updates[0].Total != 100 {
		t.Fatalf("first download update = %+v", updates[0])
	}
	if updates[1].Percentage != 100 || updates[1].Status != "正在下载镜像" || updates[1].Indeterminate {
		t.Fatalf("download complete update = %+v", updates[1])
	}
	if updates[2].Percentage != 100 || updates[2].Status != "正在下载镜像" || updates[2].Indeterminate {
		t.Fatalf("final download update = %+v", updates[2])
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

func TestPullImageRemovesTemporaryAcceleratorTag(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "settings.json"))
	if _, err := settingstore.SetHubURLs([]string{"mirror.example"}); err != nil {
		t.Fatal(err)
	}
	var requests []string
	transport := pullRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		switch {
		case strings.HasSuffix(req.URL.Path, "/images/create"):
			return pullResponse(http.StatusOK, `{"status":"Pull complete","id":"layer"}`), nil
		case strings.HasSuffix(req.URL.Path, "/tag"):
			return pullResponse(http.StatusCreated, ``), nil
		case req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/images/mirror.example/library/nginx:latest"):
			return pullResponse(http.StatusOK, `[{"Untagged":"mirror.example/library/nginx:latest"}]`), nil
		default:
			return pullResponse(http.StatusNotFound, `{"message":"not found"}`), nil
		}
	})
	cli, err := client.NewClientWithOpts(client.WithHost("http://docker.test"), client.WithVersion("1.25"), client.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	pulled, err := PullImage(context.Background(), cli, "nginx:latest", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pulled != "nginx:latest" {
		t.Fatalf("pulled reference = %q, want nginx:latest", pulled)
	}
	if len(requests) != 3 || !strings.HasPrefix(requests[0], "POST ") || !strings.HasPrefix(requests[1], "POST ") || !strings.HasPrefix(requests[2], "DELETE ") {
		t.Fatalf("request sequence = %v, want pull, tag, delete", requests)
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
