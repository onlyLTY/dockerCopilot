package utiles

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
)

func TestRemoveContainerWithClientClearsIgnoredNameAfterSuccess(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "appSettings.json"))
	if err := settingstore.SetContainerUpdateIgnored("web", true); err != nil {
		t.Fatal(err)
	}

	requests := []string{}
	client := newTestDockerClient(t, func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path)
		if req.Method == http.MethodGet && req.URL.Path == "/v1.25/containers/web-id/json" {
			return jsonResponse(http.StatusOK, `{"Name":"/web"}`), nil
		}
		if req.Method == http.MethodDelete && req.URL.Path == "/v1.25/containers/web-id" {
			return emptyResponse(http.StatusNoContent), nil
		}
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
	})

	if err := RemoveContainerWithClient(context.Background(), client, "web-id", false, ""); err != nil {
		t.Fatal(err)
	}
	if settingstore.IsContainerUpdateIgnored("web") {
		t.Fatal("ignored container setting was not removed")
	}
	if got, want := requests, []string{"GET /v1.25/containers/web-id/json", "DELETE /v1.25/containers/web-id"}; !equalStrings(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestRemoveContainerWithClientKeepsIgnoredNameWhenRemoveFails(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "appSettings.json"))
	if err := settingstore.SetContainerUpdateIgnored("web", true); err != nil {
		t.Fatal(err)
	}

	client := newTestDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet && req.URL.Path == "/v1.25/containers/web-id/json" {
			return jsonResponse(http.StatusOK, `{"Name":"/web"}`), nil
		}
		if req.Method == http.MethodDelete && req.URL.Path == "/v1.25/containers/web-id" {
			return jsonResponse(http.StatusConflict, `{"message":"container is running"}`), nil
		}
		return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
	})

	if err := RemoveContainerWithClient(context.Background(), client, "web-id", false, ""); err == nil {
		t.Fatal("remove failure was not returned")
	}
	if !settingstore.IsContainerUpdateIgnored("web") {
		t.Fatal("ignored container setting was removed after failed delete")
	}
}

func TestRemoveContainerWithClientForceStopsBeforeRemove(t *testing.T) {
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(t.TempDir(), "appSettings.json"))
	requests := []string{}
	client := newTestDockerClient(t, func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1.25/containers/web-id/json":
			return jsonResponse(http.StatusOK, `{"Name":"/web"}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1.25/containers/web-id/stop":
			if req.URL.Query().Get("t") != "10" {
				return nil, fmt.Errorf("stop timeout = %q, want 10", req.URL.Query().Get("t"))
			}
			return emptyResponse(http.StatusNoContent), nil
		case req.Method == http.MethodDelete && req.URL.Path == "/v1.25/containers/web-id":
			if req.URL.Query().Get("force") != "1" {
				return nil, fmt.Errorf("force = %q, want 1", req.URL.Query().Get("force"))
			}
			return emptyResponse(http.StatusNoContent), nil
		default:
			return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	})

	if err := RemoveContainerWithClient(context.Background(), client, "web-id", true, ""); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /v1.25/containers/web-id/json?",
		"POST /v1.25/containers/web-id/stop?t=10",
		"DELETE /v1.25/containers/web-id?force=1",
	}
	if got := requests; !equalStrings(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func newTestDockerClient(t *testing.T, transport roundTripFunc) *client.Client {
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func emptyResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
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
