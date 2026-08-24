package utiles

import "testing"

func TestIsSelfContainerIDMatchesShortAndLongIDs(t *testing.T) {
	full := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for _, id := range []string{full, full[:12], "sha256:" + full} {
		if !isSelfContainerID(id, []string{full}) {
			t.Fatalf("expected %q to match the running container", id)
		}
	}
	if isSelfContainerID("abcdefabcdef", []string{full}) {
		t.Fatal("unrelated container matched the running container")
	}
}

func TestNormalizeContainerIDExtractsSystemdScope(t *testing.T) {
	id := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if got := normalizeContainerID("docker-" + id + ".scope"); got != id {
		t.Fatalf("unexpected normalized id %q", got)
	}
}
