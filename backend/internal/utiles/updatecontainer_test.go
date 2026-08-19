package utiles

import (
	"context"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

type recordingImageTagger struct {
	imageID  string
	imageTag string
}

func (r *recordingImageTagger) ImageTag(_ context.Context, imageID, imageTag string) error {
	r.imageID = imageID
	r.imageTag = imageTag
	return nil
}

func TestPreserveOldImageTag(t *testing.T) {
	tagger := &recordingImageTagger{}
	tag, err := preserveOldImageTag(context.Background(), tagger, "sha256:a1b2c3d4e5f67890", "nginx:latest")
	if err != nil {
		t.Fatalf("preserveOldImageTag() error = %v", err)
	}
	if tag != "nginx:latest-olda1b2c3" {
		t.Fatalf("preserveOldImageTag() tag = %q, want %q", tag, "nginx:latest-olda1b2c3")
	}
	if tagger.imageID != "sha256:a1b2c3d4e5f67890" {
		t.Fatalf("ImageTag source = %q, want old image ID", tagger.imageID)
	}
	if tagger.imageTag != "nginx:latest-olda1b2c3" {
		t.Fatalf("ImageTag target = %q, want %q", tagger.imageTag, "nginx:latest-olda1b2c3")
	}
}

func TestOldImageTag(t *testing.T) {
	tests := []struct {
		name     string
		imageID  string
		imageRef string
		want     string
	}{
		{name: "latest", imageID: "sha256:a1b2c3d4e5f67890", imageRef: "nginx:latest", want: "nginx:latest-olda1b2c3"},
		{name: "default tag", imageID: "sha256:1234567890abcdef", imageRef: "nginx", want: "nginx:latest-old123456"},
		{name: "registry", imageID: "sha256:abcdef1234567890", imageRef: "registry.example.com/team/app:v2", want: "registry.example.com/team/app:v2-oldabcdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := oldImageTag(tt.imageID, tt.imageRef)
			if err != nil {
				t.Fatalf("oldImageTag() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("oldImageTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOldImageTagRejectsShortID(t *testing.T) {
	if _, err := oldImageTag("sha256:abc", "nginx:latest"); err == nil {
		t.Fatal("oldImageTag() expected an error for a short image ID")
	}
}

func TestClearPullProgress(t *testing.T) {
	progress := svc.TaskProgress{
		ProgressType:  svc.ProgressTypeImagePull,
		Indeterminate: true,
		Current:       80,
		Total:         100,
	}

	clearPullProgress(&progress)

	if progress.ProgressType != "" || progress.Indeterminate || progress.Current != 0 || progress.Total != 0 {
		t.Fatalf("pull progress metadata was not cleared: %+v", progress)
	}
}
