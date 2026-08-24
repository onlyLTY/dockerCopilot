package module

import (
	"encoding/base64"
	"strings"
	"testing"

	ref "github.com/distribution/reference"
	"github.com/docker/docker/api/types/image"
	"github.com/onlyLTY/dockerCopilot/internal/types"
)

func TestCredentialsForReferenceSupportsRegistryPort(t *testing.T) {
	auth := base64.StdEncoding.EncodeToString([]byte("user:PasswordCase"))
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"registry.example:5000":{"auth":"`+auth+`"}}}`)
	credentials, err := credentialsForReference("registry.example:5000/team/image:tag")
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Basic != auth || credentials.Encoded == "" {
		t.Fatalf("credentials were not loaded: %+v", credentials)
	}
}

func TestGetAuthURLPreservesCaseSensitiveValues(t *testing.T) {
	named, err := ref.ParseNormalizedNamed("registry.example/team/image")
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := GetAuthURL(`Bearer realm="https://auth.example/TokenPath?Existing=ABC",service="CaseSensitiveService"`, named)
	if err != nil {
		t.Fatal(err)
	}
	if authURL.Path != "/TokenPath" || authURL.Query().Get("Existing") != "ABC" || authURL.Query().Get("service") != "CaseSensitiveService" {
		t.Fatalf("challenge values changed case: %s", authURL)
	}
}

func TestExpandImageReferencesKeepsTagsIndependent(t *testing.T) {
	expanded := expandImageReferences([]types.Image{{Summary: image.Summary{
		ID: "sha256:same", RepoTags: []string{"example/repo:stable", "example/repo:latest"},
	}}})
	if len(expanded) != 2 || expanded[0].Reference == expanded[1].Reference {
		t.Fatalf("tags were not expanded independently: %+v", expanded)
	}
}

func TestRepoDigestsAreFilteredByRepository(t *testing.T) {
	digests := repoDigestsForReference([]string{
		"docker.io/library/postgres@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"mirror.example/postgres@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}, "postgres:17-alpine")
	if len(digests) != 1 || !strings.Contains(digests[0], "sha256:aaaa") {
		t.Fatalf("unexpected repository digests: %v", digests)
	}
}
