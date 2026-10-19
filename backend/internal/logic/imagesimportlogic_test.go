package logic

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestReadImageLoadStreamCollectsLoadedImages(t *testing.T) {
	body := strings.Join([]string{
		`{"stream":"Loaded image: private/app:1.0\n"}`,
		`{"stream":"Loaded image: private/app:latest\n"}`,
		`{"stream":"Loaded image: private/app:1.0\n"}`,
		``,
	}, "\n")
	loaded, err := readImageLoadStream(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded = %v, want 2 unique refs", loaded)
	}
	if loaded[0] != "private/app:1.0" || loaded[1] != "private/app:latest" {
		t.Fatalf("loaded = %v, want refs without Loaded image prefix", loaded)
	}
}

func TestReadImageLoadStreamReturnsDockerError(t *testing.T) {
	body := `{"errorDetail":{"message":"invalid tar header"},"error":"invalid tar header"}`
	if _, err := readImageLoadStream(strings.NewReader(body)); err == nil {
		t.Fatal("expected error for errorDetail response")
	}
}

func TestDecompressImageArchiveHandlesGzip(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte("tar-content"))
	_ = writer.Close()

	reader, closeReader, err := decompressImageArchive(bytes.NewReader(compressed.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer closeReader()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "tar-content" {
		t.Fatalf("decompressed = %q, want tar-content", content)
	}
}

func newImportTestContext(t *testing.T, dockerClient *client.Client) *svc.ServiceContext {
	t.Helper()
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	t.Setenv("TASK_PROGRESS_PATH", filepath.Join(root, "tasks.json"))
	svcCtx := svc.NewServiceContext(config.Config{})
	svcCtx.DockerClient = dockerClient
	return svcCtx
}

func waitImportTask(t *testing.T, svcCtx *svc.ServiceContext, taskID string) svc.TaskProgress {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		progress, ok := svcCtx.GetProgress(taskID)
		if ok && progress.IsDone {
			return progress
		}
		time.Sleep(10 * time.Millisecond)
	}
	progress, _ := svcCtx.GetProgress(taskID)
	t.Fatalf("task %s did not finish: %+v", taskID, progress)
	return svc.TaskProgress{}
}

func TestSubmitImportLoadsArchiveInBackground(t *testing.T) {
	var received []byte
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/v1.25/images/load" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		received, _ = io.ReadAll(req.Body)
		body := `{"stream":"Loaded image: private/app:1.0\n"}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
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

	svcCtx := newImportTestContext(t, dockerClient)
	archive := []byte("fake-tar-bytes")
	resp, err := NewImportImagesLogic(context.Background(), svcCtx).SubmitImport(ImportInput{
		Filename: "private-app.tar",
		Reader:   bytes.NewReader(archive),
		Total:    int64(len(archive)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d (%s), want 200", resp.Code, resp.Msg)
	}
	data, ok := resp.Data.(importSubmitResult)
	if !ok || data.TaskID == "" {
		t.Fatalf("response data = %#v, want submitted task id", resp.Data)
	}

	progress := waitImportTask(t, svcCtx, data.TaskID)
	if progress.Failed {
		t.Fatalf("import failed: %+v", progress)
	}
	if progress.Percentage != 100 || progress.Message != importDoneMessage {
		t.Fatalf("progress = %+v, want done at 100%%", progress)
	}
	if !strings.Contains(progress.DetailMsg, "private/app:1.0") {
		t.Fatalf("detail = %q, want loaded image ref", progress.DetailMsg)
	}
	if !bytes.Equal(received, archive) {
		t.Fatalf("docker received %q, want archive bytes", received)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(mustGlob(t, filepath.Join(datadir.TempDir(), importTempFilePrefix+"*"))) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if leftovers := mustGlob(t, filepath.Join(datadir.TempDir(), importTempFilePrefix+"*")); len(leftovers) > 0 {
		t.Fatalf("temporary archive was not removed: %v", leftovers)
	}
}

func TestSubmitImportMarksTaskFailedWhenLoadFails(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"errorDetail":{"message":"invalid tar header"},"error":"invalid tar header"}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
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

	svcCtx := newImportTestContext(t, dockerClient)
	resp, err := NewImportImagesLogic(context.Background(), svcCtx).SubmitImport(ImportInput{
		Filename: "broken.tar",
		Reader:   bytes.NewReader([]byte("broken")),
		Total:    6,
	})
	if err != nil || resp.Code != 200 {
		t.Fatalf("submit = %#v, err = %v", resp, err)
	}
	taskID := resp.Data.(importSubmitResult).TaskID
	progress := waitImportTask(t, svcCtx, taskID)
	if !progress.Failed || progress.Message != importFailedMessage {
		t.Fatalf("progress = %+v, want failed import", progress)
	}
	if !strings.Contains(progress.DetailMsg, "格式不正确") {
		t.Fatalf("detail = %q, want archive format hint", progress.DetailMsg)
	}
}

func TestSubmitImportTooLargeMessage(t *testing.T) {
	svcCtx := newImportTestContext(t, nil)
	svcCtx.Config = config.Config{ImageImportMaxBytes: 1024}
	resp, err := NewImportImagesLogic(context.Background(), svcCtx).SubmitImport(ImportInput{
		Filename: "too-large.tar",
		Reader:   bytes.NewReader([]byte("x")),
	})
	if err == nil {
		t.Fatal("expected docker unavailable error when client is nil")
	}
	if resp.Code != 503 {
		t.Fatalf("response code = %d, want docker unavailable 503 when client is nil", resp.Code)
	}

	logic := NewImportImagesLogic(context.Background(), svcCtx)
	resp, err = logic.uploadFailure(context.Background(), "too-large", "导入镜像 x.tar", &http.MaxBytesError{Limit: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 413 || !strings.Contains(resp.Msg, "大小限制") {
		t.Fatalf("response = %#v, want 413 size limit message", resp)
	}
	progress, ok := svcCtx.GetProgress("too-large")
	if !ok || !progress.Failed || !progress.IsDone {
		t.Fatalf("progress = %+v, want failed task", progress)
	}
}

func TestImportResultJSONShape(t *testing.T) {
	content, err := json.Marshal(importSubmitResult{TaskID: "t1", Filename: "a.tar"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"taskID":"t1"`) || !strings.Contains(string(content), `"filename":"a.tar"`) {
		t.Fatalf("json = %s", content)
	}
}

func mustGlob(t *testing.T, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestCleanupStaleImageImports(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DATA_DIR", root)
	if err := os.MkdirAll(datadir.TempDir(), 0755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(datadir.TempDir(), importTempFilePrefix+"stale.tar")
	if err := os.WriteFile(stale, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(datadir.TempDir(), importTempFilePrefix+"fresh.tar")
	if err := os.WriteFile(fresh, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	CleanupStaleImageImports(10 * time.Minute)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale archive still exists: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("fresh archive was removed: %v", err)
	}
}
