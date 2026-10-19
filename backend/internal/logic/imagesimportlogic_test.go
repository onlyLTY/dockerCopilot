package logic

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/config"
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

func TestImportImagesLoadsArchive(t *testing.T) {
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

	svcCtx := &svc.ServiceContext{DockerClient: dockerClient}
	archive := []byte("fake-tar-bytes")
	resp, err := NewImportImagesLogic(context.Background(), svcCtx).Import(ImportInput{
		Filename: "private-app.tar",
		Reader:   bytes.NewReader(archive),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 {
		t.Fatalf("response code = %d (%s), want 200", resp.Code, resp.Msg)
	}
	if !bytes.Equal(received, archive) {
		t.Fatalf("docker received %q, want archive bytes", received)
	}
	data, ok := resp.Data.(importResult)
	if !ok {
		t.Fatalf("response data type = %T, want importResult", resp.Data)
	}
	if data.Count != 1 || len(data.Loaded) != 1 || data.Loaded[0] != "private/app:1.0" {
		t.Fatalf("import result = %#v, want one loaded image", data)
	}
}

func TestImportImagesTooLargeMessage(t *testing.T) {
	svcCtx := &svc.ServiceContext{Config: config.Config{ImageImportMaxBytes: 1024}}
	resp, err := NewImportImagesLogic(context.Background(), svcCtx).Import(ImportInput{
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
	resp, err = logic.importFailure(&http.MaxBytesError{Limit: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code != 413 || !strings.Contains(resp.Msg, "大小限制") {
		t.Fatalf("response = %#v, want 413 size limit message", resp)
	}
}

func TestImportResultJSONShape(t *testing.T) {
	content, err := json.Marshal(importResult{Loaded: []string{"a"}, Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"loaded":["a"]`) || !strings.Contains(string(content), `"count":1`) {
		t.Fatalf("json = %s", content)
	}
}
