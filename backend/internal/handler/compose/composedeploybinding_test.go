package compose

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestDeployPreviewRequestBindsProjectIDFromPath(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compose/projects/project-a/deploy/preview",
		strings.NewReader(`{"filename":"compose.yaml"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req = pathvar.WithVars(req, map[string]string{"id": "project-a"})

	var parsed types.ComposeDeployPreviewReq
	if err := httpx.Parse(req, &parsed); err != nil {
		t.Fatalf("httpx.Parse() error = %v", err)
	}
	if parsed.ProjectID != "project-a" {
		t.Fatalf("ProjectID = %q, want %q", parsed.ProjectID, "project-a")
	}
	if parsed.Filename != "compose.yaml" {
		t.Fatalf("Filename = %q, want %q", parsed.Filename, "compose.yaml")
	}
}

func TestDeployPreviewRequestRequiresPathProjectID(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "filename only", body: `{"filename":"compose.yaml"}`},
		{name: "body project id cannot replace path", body: `{"projectId":"project-a","filename":"compose.yaml"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/compose/projects//deploy/preview",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			var parsed types.ComposeDeployPreviewReq
			if err := httpx.Parse(req, &parsed); err == nil {
				t.Fatal("httpx.Parse() error = nil, want missing path id error")
			}
		})
	}
}

func TestDeployRequestBindsProjectIDFromPath(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/compose/projects/project-a/deploy",
		strings.NewReader(`{"filename":"compose.yaml","confirmToken":"token","confirmWarnings":true,"pullImages":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req = pathvar.WithVars(req, map[string]string{"id": "project-a"})

	var parsed types.ComposeDeployReq
	if err := httpx.Parse(req, &parsed); err != nil {
		t.Fatalf("httpx.Parse() error = %v", err)
	}
	if parsed.ProjectID != "project-a" {
		t.Fatalf("ProjectID = %q, want %q", parsed.ProjectID, "project-a")
	}
	if parsed.Filename != "compose.yaml" || parsed.ConfirmToken != "token" || !parsed.ConfirmWarnings || !parsed.PullImages {
		t.Fatalf("unexpected parsed request: %+v", parsed)
	}
}
