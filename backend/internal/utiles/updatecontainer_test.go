package utiles

import (
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

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
