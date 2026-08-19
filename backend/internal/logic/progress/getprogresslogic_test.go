package progress

import (
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestProgressDataIncludesTaskTiming(t *testing.T) {
	data := progressData(svc.TaskProgress{
		TaskID:     "task-1",
		Refresh:    true,
		StartedAt:  100,
		EndedAt:    250,
		DurationMs: 150,
		Steps: []svc.TaskStep{{
			Message:    "完成",
			StartedAt:  110,
			EndedAt:    240,
			DurationMs: 130,
		}},
	})

	if data["refresh"] != true || data["startedAt"] != int64(100) ||
		data["endedAt"] != int64(250) || data["durationMs"] != int64(150) {
		t.Fatalf("task timing fields were not mapped: %#v", data)
	}
	if len(data["steps"].([]svc.TaskStep)) != 1 {
		t.Fatalf("steps were not mapped: %#v", data["steps"])
	}
}
