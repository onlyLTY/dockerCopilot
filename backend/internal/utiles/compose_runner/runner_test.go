package compose_runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/client"
)

// deployService 期望删除并重建同名容器，而不是因配置哈希相同跳过；
// 未勾选拉取时也不应访问镜像仓库。
func TestDeployServiceRecreatesExistingContainerWithoutPull(t *testing.T) {
	var requests []string
	dockerClient := newRunnerTestClient(t, func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		switch {
		// imageExists
		case req.Method == http.MethodGet && req.URL.Path == "/v1.25/images/web:1.0/json":
			return runnerJSONResponse(http.StatusOK, `{"Id":"sha256:new"}`), nil
		// findContainerByName
		case req.Method == http.MethodGet && req.URL.Path == "/v1.25/containers/json":
			if got := req.URL.Query().Get("filters"); !strings.Contains(got, `^/demo-web-1$`) {
				return nil, fmt.Errorf("unexpected container filter: %s", got)
			}
			return runnerJSONResponse(http.StatusOK, `[{"Id":"old-id","Names":["/demo-web-1"],"State":"running"}]`), nil
		// removeContainer -> inspect name -> stop -> delete
		case req.Method == http.MethodGet && req.URL.Path == "/v1.25/containers/old-id/json":
			return runnerJSONResponse(http.StatusOK, `{"Name":"/demo-web-1"}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1.25/containers/old-id/stop":
			return runnerEmptyResponse(http.StatusNoContent), nil
		case req.Method == http.MethodDelete && req.URL.Path == "/v1.25/containers/old-id":
			return runnerEmptyResponse(http.StatusNoContent), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1.25/containers/create":
			return runnerJSONResponse(http.StatusCreated, `{"Id":"new-id"}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1.25/containers/new-id/start":
			return runnerEmptyResponse(http.StatusNoContent), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	svc := composeTypes.ServiceConfig{Image: "web:1.0", ContainerName: "demo-web-1"}
	err := deployService(
		context.Background(), dockerClient, "demo", t.TempDir(), "web", svc,
		"demo_default", map[string]string{}, map[string]string{}, nil,
		false, time.Minute, func(string, ...interface{}) {}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(requests, "\n")
	if !strings.Contains(joined, "DELETE /v1.25/containers/old-id") {
		t.Fatalf("existing container was not removed, requests:\n%s", joined)
	}
	if !strings.Contains(joined, "POST /v1.25/containers/create") ||
		!strings.Contains(joined, "POST /v1.25/containers/new-id/start") {
		t.Fatalf("container was not recreated, requests:\n%s", joined)
	}
	if strings.Contains(joined, "/images/create") {
		t.Fatalf("image pull must not happen without pullImages, requests:\n%s", joined)
	}
}

func newRunnerTestClient(t *testing.T, transport roundTripFunc) *client.Client {
	t.Helper()
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost("http://docker.test"),
		client.WithVersion("1.25"),
		client.WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dockerClient.Close() })
	return dockerClient
}

func runnerJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func runnerEmptyResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
