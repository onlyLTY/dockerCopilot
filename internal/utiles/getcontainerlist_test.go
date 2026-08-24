package utiles

import (
	"testing"

	"github.com/docker/docker/api/types/image"
)

func TestContainerNeedsImageUpdateUsesRunningImageID(t *testing.T) {
	if !containerNeedsImageUpdate(true, "sha256:new", "sha256:old") {
		t.Fatal("remote digest change must be reported")
	}
	if !containerNeedsImageUpdate(false, "sha256:new", "sha256:old") {
		t.Fatal("container still running the old image must be reported")
	}
	if containerNeedsImageUpdate(false, "sha256:new", "sha256:new") {
		t.Fatal("container running the tag-resolved image must be current")
	}
	if containerNeedsImageUpdate(false, "", "sha256:old") {
		t.Fatal("an unresolved tag must not create a false update")
	}
}

func TestImageIDsByReferenceNormalizesAllTags(t *testing.T) {
	resolved := imageIDsByReference([]image.Summary{
		{ID: "sha256:postgres", RepoTags: []string{"postgres:17-alpine", "docker.io/library/postgres:latest"}},
		{ID: "sha256:private", RepoTags: []string{"registry.example:5000/team/app:v1"}},
	})
	if resolved["docker.io/library/postgres:17-alpine"] != "sha256:postgres" {
		t.Fatalf("Docker Hub tag was not normalized: %+v", resolved)
	}
	if resolved["registry.example:5000/team/app:v1"] != "sha256:private" {
		t.Fatalf("private registry tag was not normalized: %+v", resolved)
	}
}
