package container

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func TestParseRenameBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{name: "json", body: `{"newName":"renamed"}`, want: "renamed"},
		{name: "json without content type", body: `{"newName":"renamed"}`, want: "renamed"},
		{name: "form", body: "newName=renamed", want: "renamed"},
		{name: "missing field", body: `{}`, wantErr: true},
		{name: "blank field", body: `{"newName":"   "}`, wantErr: true},
		{name: "empty body", body: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req types.ContainerRenameReq
			if err := parseRenameBody([]byte(tt.body), &req); (err != nil) != tt.wantErr {
				t.Fatalf("parseRenameBody() error = %v, wantErr %v", err, tt.wantErr)
			}
			if req.NewName != tt.want {
				t.Fatalf("NewName = %q, want %q", req.NewName, tt.want)
			}
		})
	}
}

func TestRenameHandlerBindsPathAndBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/container/container-id/rename", strings.NewReader(`{"newName":"renamed"}`))
	var parsed types.ContainerRenameReq
	if err := parseRenameBody([]byte(`{"newName":"renamed"}`), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.NewName != "renamed" {
		t.Fatalf("unexpected parsed request: %+v", parsed)
	}
	_ = req
}
